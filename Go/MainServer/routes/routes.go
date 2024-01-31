package routes

import (
	"MainServer/config"
	"MainServer/geocoding"
	"context"
	"crypto/tls"
	"log"
	"os"
	"time"

	routespb "cloud.google.com/go/maps/routing/apiv2/routingpb"
	"github.com/sony/gobreaker"
	"google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	fieldMask  = "*"
	serverAddr = "routes.googleapis.com:443"
)

var breaker, breakerGoogle *gobreaker.CircuitBreaker

func init() {
	log.SetOutput(os.Stdout)
	// Inizializza il circuit breaker
	breaker = config.GetBreakerSettings("BreakerConnectionGoogle")
	breakerGoogle = config.GetBreakerSettings("BreakerExecutionGoogle")
}

func executeConnectionBreaker(breaker *gobreaker.CircuitBreaker, configs *tls.Config) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	var err error

	_, execErr := breaker.Execute(func() (interface{}, error) {
		conn, err = grpc.Dial(serverAddr, grpc.WithTransportCredentials(credentials.NewTLS(configs)))

		if err != nil {
			log.Printf("Errore connessione a Connessione Google: %v", err)
			return nil, err
		}

		return conn, nil
	})

	if execErr != nil {
		// Se il circuito è aperto, execErr sarà gobreaker.ErrOpenState
		return nil, execErr
	}

	return conn, nil
}

func executeSendConnectionBreaker(breaker *gobreaker.CircuitBreaker, ctx context.Context, req *routespb.ComputeRoutesRequest, client routespb.RoutesClient) (*routespb.ComputeRoutesResponse, error) {

	var resp *routespb.ComputeRoutesResponse
	var err error

	_, execErr := breaker.Execute(func() (interface{}, error) {

		resp, err = client.ComputeRoutes(ctx, req)

		if err != nil {
			log.Printf("INTERNO: Errore connessione a Api Routes: %v", err)
			return nil, err
		}

		return resp, nil
	})

	if execErr != nil {
		// Se il circuito è aperto, execErr sarà gobreaker.ErrOpenState
		log.Printf("ESTERNO: Errore connessione a Api Routes: %v", execErr)
		return nil, execErr
	}

	return resp, nil
}

func Routing(partenza string, arrivo string) *routespb.ComputeRoutesResponse {

	configs := tls.Config{}
	conn, err := executeConnectionBreaker(breaker, &configs)

	for breaker.State().String() == gobreaker.StateClosed.String() && err != nil {

		conn, err = executeConnectionBreaker(breaker, &configs)
		log.Println("Riprova Connessione Google")
	}

	if breaker.State().String() != gobreaker.StateClosed.String() {

		log.Println("Stato Aperto " + breaker.Name())
		log.Printf("Errore %s ", err)
		return nil
	}

	if conn != nil {
		defer conn.Close()
	}
	if err != nil {
		log.Printf("Connessione non riuscita: %v", err)
	}

	apiKey, err := config.GetParametroFromConfig("apigoogletoken")
	if err != nil {
		log.Println(err)
	}
	client := routespb.NewRoutesClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	ctx = metadata.AppendToOutgoingContext(ctx, "X-Goog-Api-Key", apiKey)
	ctx = metadata.AppendToOutgoingContext(ctx, "X-Goog-Fieldmask", fieldMask)
	defer cancel()

	// crea latitudine e longitudine dalla funzione di geocoding -> posizione in stringa

	lat_par, lon_par := geocoding.StartUpConnection(partenza)

	lat_arr, lon_arr := geocoding.StartUpConnection(arrivo)

	if lat_par == 0 || lon_par == 0 {
		log.Printf("Errore Posizione Partenza")
		return nil
	}

	if lat_arr == 0 || lon_arr == 0 {
		log.Printf("Errore Posizione Arrivo")
		return nil
	}
	// crea l'origine utilizzando latitudine e longitudine
	origin := &routespb.Waypoint{
		LocationType: &routespb.Waypoint_Location{
			Location: &routespb.Location{
				LatLng: &latlng.LatLng{
					Latitude:  lat_par,
					Longitude: lon_par,
				},
			},
		},
	}

	// crea la destinazione utilizzando latitudine e longitudine
	destination := &routespb.Waypoint{
		LocationType: &routespb.Waypoint_Location{
			Location: &routespb.Location{
				LatLng: &latlng.LatLng{
					Latitude:  lat_arr,
					Longitude: lon_arr,
				},
			},
		},
	}

	req := &routespb.ComputeRoutesRequest{
		Origin:                   origin,
		Destination:              destination,
		TravelMode:               routespb.RouteTravelMode_DRIVE,
		RoutingPreference:        routespb.RoutingPreference_TRAFFIC_AWARE_OPTIMAL,
		ComputeAlternativeRoutes: false,
		Units:                    routespb.Units_METRIC,
		RouteModifiers: &routespb.RouteModifiers{
			AvoidTolls:    false,
			AvoidHighways: false,
			AvoidFerries:  true,
		},
		PolylineQuality: routespb.PolylineQuality_OVERVIEW,
	}

	// esegui rpc
	resp, err := executeSendConnectionBreaker(breakerGoogle, ctx, req, client)

	log.Println("Invio richiesta Google")
	for breakerGoogle.State().String() == gobreaker.StateClosed.String() && err != nil {
		resp, err = executeSendConnectionBreaker(breakerGoogle, ctx, req, client)
		log.Println("Riprova Invio richiesta Google")
	}

	if breakerGoogle.State().String() != gobreaker.StateClosed.String() {

		log.Println("Stato Aperto " + breaker.Name())
		log.Printf("Errore %s ", err)
		return nil
	}

	if err != nil {
		// "rpc error: code = InvalidArgument desc = Request contains an invalid
		// argument" potrebbe indicare che il tuo progetto non ha accesso a Routes
		log.Printf("%s", err)
		return nil
	}

	return resp
}
