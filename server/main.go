package main

import (
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"net"

	"acos/protocol"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Ошибка: ", err)
	}
	fmt.Println("Сервер слушает на порту 8080")

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Ошибка: ", err)
			continue
		}
		fmt.Println("Воркер подключился")
		handleWorker(conn)
	}
}

func handleWorker(conn net.Conn) {
	defer conn.Close()
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	task := protocol.Task{
		ID:      1,
		Payload: []byte("Ping"),
	}

	fmt.Println("Отправка Ping")
	err := encoder.Encode(task)
	if err != nil {
		fmt.Println("Данные не ушли:", err)
		return
	}

	var res protocol.Result
	err = decoder.Decode(&res)
	if err != nil {
		fmt.Println("Ошибка при получении ответа:", err)
		return
	}
	fmt.Printf("Воркер #%d прислал данные: %s\n", res.ID, res.Payload)
	fmt.Printf("Передача завершена")
}