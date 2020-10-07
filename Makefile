DATA:=data
PORT:=8000

run:
	go run main.go -data=${DATA} -port=${PORT}
