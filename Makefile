DATA:=data
PORT:=8000

start:
	go run main.go -data=${DATA} -port=${PORT}
