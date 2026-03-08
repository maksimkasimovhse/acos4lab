package main

import (
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net"

	"acos/protocol"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatal("Не удалось соединиться с сервером: ", err)
	}
	defer conn.Close()
	fmt.Println("Успешное подключение к серверу")

	decoder := json.NewDecoder(conn)
	var task protocol.Task
	err = decoder.Decode(&task)
	if err != nil {
		fmt.Println("Ошибка дешифрования", err)
		return
	}
	fmt.Printf("Получена задача #%d с данными %s\n", task.ID, string(task.Payload))

	encoder := json.NewEncoder(conn)
	res := protocol.Result{
		ID:      task.ID,
		Success: true,
		Payload: []byte("Pong"),
	}
	err = encoder.Encode(res)
	if err != nil {
		fmt.Println("Ошибка отправки ответа: ", err)
	}

}
