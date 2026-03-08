package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"net"
	"os"

	"acos/protocol"
)

const (
	Port           = ":8080"
	RowsPerTask    = 100
	BufferCapacity = 100
)

func main() {
	jobs := make(chan protocol.Task, BufferCapacity)
	retry := make(chan protocol.Task, BufferCapacity)
	results := make(chan protocol.Result, BufferCapacity) //динамически высчитывать размер буфера

	ln, err := net.Listen("tcp", Port)
	if err != nil {
		fmt.Println("Ошибка: ", err)
	}
	fmt.Println("Сервер слушает на порту 8080")

	go CuttingAndDistribution(jobs)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Ошибка подключения к воркеру: ", err)
			continue
		}
		fmt.Println("Воркер подключился")
		go handleWorker(conn, jobs, retry, results)
	}
}

func handleWorker(conn net.Conn, jobs, retry, results chan protocol.Task) {
	defer conn.Close()
	encoder := json.NewEncoder(conn)
	for {
		task := getTask(jobs, retry)
		err := encoder.Encode(task)
		if err != nil {
			retry <- task
		}
	}
}

func getTask(jobs, retry chan protocol.Task) protocol.Task {
	select {
	case task := <-retry:
		return task
	default:

	}

	select {
	case task := <-retry:
		return task
	case task := <-jobs:
		return task
	}

}
func CuttingAndDistribution(jobs chan protocol.Task) {
	filePath := "images/input.png"
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Ошибка: не удалось найти файл по пути %s: %v\n", filePath, err)
		return
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		fmt.Printf("Ошибка декодирования: %v\n", err)
		return
	}
	fmt.Printf("Файл открыт. Формат: %s, Размер: %dx%d\n", format, img.Bounds().Dx(), img.Bounds().Dy())
	bounds := img.Bounds()
	rgbaImg := image.NewRGBA(bounds)
	draw.Draw(rgbaImg, bounds, img, bounds.Min, draw.Src)

	rowsPerTask := RowsPerTask
	stride := rgbaImg.Stride
	bytesPerTask := rowsPerTask * stride

	for i := 0; i < bounds.Dy(); i += rowsPerTask {
		start := i * stride
		end := start + bytesPerTask
		if end > len(rgbaImg.Pix) {
			end = len(rgbaImg.Pix)
		}

		task := protocol.Task{
			ID:      i / rowsPerTask,
			Payload: rgbaImg.Pix[start:end],
			Bounds:  image.Rect(0, i, bounds.Dx(), i+rowsPerTask),
		}

		jobs <- task
	}
}
