import json
from datetime import datetime, timedelta
from flask import Flask, request, abort, jsonify
from prometheus_api_client import PrometheusConnect
from DB import ConnectionDB
from model import reevaluate_model
from probability_estimation import compute_out_of_bounds_probability

app = Flask(__name__)


def sla_manager(viol_max, current_value):
    return viol_max < float(current_value)


@app.route('/create_update', methods=['PUT'])
def put_metrics():
    data = request.json

    metric_name = data.get('metric')
    viol_max = data.get('viol_max')

    connectionDB = connectionDB_obj.create_server_connection()

    res = connectionDB_obj.check_metrics(connectionDB, f"SELECT name FROM metrics WHERE name ='{metric_name}'")

    if res[0] == "ok":
        try:
            time = str(datetime.now().strftime('%Y-%m-%d %H:%M:%S'))
            connectionDB_obj.do_query(connectionDB,
                                      f"UPDATE metrics SET viol_max = {viol_max}, time = '{time}' WHERE name = '{metric_name}'")
            return jsonify({"message": "SLA metrics updated successfully"})
        except Exception as e:
            print(f"Error: '{e}'")
            abort(500, description="Internal Server Error")
    else:
        try:
            time = str(datetime.now().strftime('%Y-%m-%d %H:%M:%S'))
            connectionDB_obj.do_query(connectionDB,
                                      f"INSERT INTO metrics (name, viol_max, time) VALUES ('{metric_name}', {viol_max}, '{time}')")
            return jsonify({"message": "SLA metrics created successfully"})
        except Exception as e:
            print(f"Error: '{e}'")
            abort(500, description="Internal Server Error")


@app.route('/actual_value', methods=['GET'])
def actual_value():
    data = request.json
    name = data['name']

    connectionDB = connectionDB_obj.create_server_connection()

    res, row = ConnectionDB.check_metrics(connectionDB, f"SELECT * FROM metrics WHERE name ='{name}'")

    if res == "ok":
        query_result = prom.custom_query(query=name)
        current_value = query_result[0]["value"]
        violation = sla_manager(row[2], current_value[1])

        return jsonify({"metric_name": row[1], "current_value": current_value[1], "violation": violation,
                        "created_at": row[3]})
    else:
        abort(404, "Metrics not found")


@app.route('/remove_sla_metrics', methods=['DELETE'])
def remove_sla_metrics():
    name = request.args.get('name')

    connectionDB = connectionDB_obj.create_server_connection()

    res, row = ConnectionDB.do_query(connectionDB, f"DELETE FROM metrics WHERE name = '{name}'")

    if res == "ok":
        return jsonify({f"metrica eliminata: {name}"})
    else:
        abort(404, "Metrics not found")


@app.route('/violations', methods=['GET'])
def query_violations():
    data = request.json
    name = data['name']
    time = int(data['hours'])

    end_time = datetime.now()
    start_time = end_time - timedelta(hours=time)

    prom_data = prom.custom_query_range(name, start_time, end_time, "100")

    connectionDB = connectionDB_obj.create_server_connection()

    values = [x[1] for x in prom_data[0]["values"]]
    res, row = ConnectionDB.check_metrics(connectionDB, f"SELECT viol_max FROM metrics WHERE name ='{name}'")

    if res != "ok":
        abort(404, description="Metrics not found")

    violations = [value for value in values if sla_manager(row[0], value)]
    return jsonify({"violations_count": len(violations)})


@app.route('/reevaluate_model', methods=['POST'])
def reevaluate_model_endpoint():
    data = request.json
    name = data['name']
    range_in_minute = int(data['range_in_minute'])

    end_time = datetime.now()
    start_time = end_time - timedelta(minutes=range_in_minute)

    connectionDB = connectionDB_obj.create_server_connection()

    if not name or not range_in_minute:
        abort(400)

    time_series_data = prom.custom_query_range(name, start_time, end_time, "30")

    if time_series_data is None:
        abort(404, description=f"No time series data found for metric '{name}'")

    values = [x for x in time_series_data[0]["values"]]
    poly_coef_, period_coef_, error_std = reevaluate_model(values)
    trend_dill = {"poly_coef_": poly_coef_, "period_coef_": period_coef_}
    trend = json.dumps(trend_dill).replace('"', '\\"')

    try:
        res, model = ConnectionDB.check_metrics(connectionDB, f"SELECT * FROM serie_temp WHERE name ='{name}'")

        if res == "ok":
            ConnectionDB.do_query(connectionDB,
                                  f"UPDATE serie_temp SET error_std = {error_std},trend_function= \"{trend}\"  WHERE name = '{name}'")
        else:
            ConnectionDB.do_query(connectionDB,
                                  f"INSERT INTO serie_temp (name, error_std, trend_function) VALUES ('{name}', {error_std}, \"{trend}\" )")
    except Exception as e:
        abort(500, description=f"Error storing the new model for metric '{name}': {str(e)}")

    return jsonify({"message": f"Model re-evaluation completed for metric '{name}'"}), 200


@app.route('/probability', methods=['GET'])
def violation_probability():
    data = request.json
    name = data.get('name')
    time = int(data.get('minutes'))

    connectionDB = connectionDB_obj.create_server_connection()

    res, row = ConnectionDB.check_metrics(connectionDB, f"SELECT * FROM metrics WHERE name ='{name}'")

    if res != "ok":
        abort(404, description="Metrics not found")

    y_lower_bound = row[2]

    current_datetime = datetime.now()
    x_lower_limit = int(current_datetime.timestamp())
    future_datetime = current_datetime + timedelta(minutes=time)
    x_upper_limit = int(future_datetime.timestamp())

    res, model = ConnectionDB.check_metrics(connectionDB, f"SELECT * FROM serie_temp WHERE name ='{name}'")
    if res != "ok":
        abort(404, description="Metrics not found")

    if model[3]:
        model_load = json.loads(model[3])
        poly_coef_ = model_load["poly_coef_"]
        period_coef_ = model_load["period_coef_"]
    else:
        poly_coef_ = None
        period_coef_ = None

    error_std = model[2]

    try:
        probability, _, _, _ = compute_out_of_bounds_probability(poly_coef_, period_coef_, error_std, x_lower_limit,
                                                                 x_upper_limit, y_lower_bound)
        print(f"Probability of y > {y_lower_bound} for {x_lower_limit} < x < {x_upper_limit} is {probability}%")

        return jsonify({"probability": probability})

    except Exception as e:
        print(f"Error: '{e}'")
        abort(500, description="Internal Server Error")


# Punto d'ingresso per l'esecuzione dell'app Flask
if __name__ == '__main__':
    # Caricamento delle configurazioni dal file JSON
    with open("config.json", 'r') as file:
        file_content = file.read()

    data = json.loads(file_content)

    host = data['host_name']
    port = data['port']
    user = data['user']
    psw = data['password']
    db = data['database']

    prometheus_url = data['prometheus_url']
    prom = PrometheusConnect(url=prometheus_url)

    connectionDB_obj = ConnectionDB(host, port, user, psw, db)

    sla_server_host = data['sla_server_host']
    sla_server_port = data['sla_server_port']

    app.run(sla_server_host, sla_server_port, debug=True, use_reloader=False)
