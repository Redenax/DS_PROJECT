import mysql.connector
from mysql.connector import Error


class ConnectionDB:
    def __init__(self, host_name, port,user_name, user_password, db_name):
        self.host_name = host_name
        self.port = port
        self.user_name = user_name
        self.user_password = user_password
        self.db_name = db_name

    def create_server_connection(self):
        connection = None
        try:
            connection = mysql.connector.connect(
                host=self.host_name,
                port=self.port,
                user=self.user_name,
                password=self.user_password,
                database=self.db_name
            )
            print("MySQL Database connection successful")
        except Error as err:
            print(f"Error: '{err}'")

        return connection

    @staticmethod
    def do_query(connection, query):
        cursor = connection.cursor()
        try:
            cursor.execute(query)
            connection.commit()
            print("Query successful")
            return "ok"
        except Error as err:
            print(f"Error: '{err}'")
            return err
        finally:
            cursor.close()
            connection.close()

    @staticmethod
    def check_metrics(connection, query):
        cursor = connection.cursor()
        try:
            cursor.execute(query)
            result = cursor.fetchone()
            if result:
                return "ok", result
            else:
                return result,"not found"
        except Error as err:
            print(f"Error: '{err}'")
        finally:
            cursor.close()
