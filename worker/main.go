package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	"acos/protocol"
)

func main() {
	addr := os.Getenv("SERVER_ADDR")

	if addr == "" {
		addr = "localhost:8080"
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal("Не удалось соединиться с сервером: ", err)
	}
	defer conn.Close()
	fmt.Println("Успешное подключение к серверу")

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var task protocol.Task
		err := decoder.Decode(&task)
		if err != nil {
			fmt.Println("Соединение разорвано или ошибка JSON: ", err)
			break
		}
		fmt.Printf("Получена задача с ID: %d\n", task.ID)

		Result := protocol.Result{
			ID:      task.ID,
			Payload: make([]byte, len(task.Payload)),
			Bounds:  task.Bounds,
		}

		for i := 0; i < len(task.Payload); i++ {
			if (i+1)%4 == 0 {
				Result.Payload[i] = task.Payload[i]
			} else {
				Result.Payload[i] = 255 - task.Payload[i]
			}
		}

		err = encoder.Encode(Result)
		if err != nil {
			fmt.Println("Ошибка отправки ответа: ", err)
			break
		}
	}

}
