DATA:=data
PORT:=8000

run:
	go run cmd/main.go -data=${DATA} -port=${PORT}
