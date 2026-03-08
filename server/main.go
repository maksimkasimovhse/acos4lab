package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
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

	totalTasks, rgbaImg := CuttingAndDistribution(jobs)
	if rgbaImg == nil {
		fmt.Println("Критическая ошибка: невозможно подготовить изображение.")
		return
	}

	go ImageCollector(results, rgbaImg, totalTasks)

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

func handleWorker(conn net.Conn, jobs, retry chan protocol.Task, results chan protocol.Result) {
	defer conn.Close()
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	for {
		task := getTask(jobs, retry)
		err := encoder.Encode(task)
		if err != nil {
			fmt.Println("Воркер отвалился при отправке. Возвращаем задачу...")
			retry <- task
			return
		}

		var res protocol.Result
		err = decoder.Decode(&res)
		if err != nil {
			fmt.Println("Воркер отвалился при ожидании ответа. Возвращаем задачу...")
			retry <- task
			return
		}
		results <- res
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
func CuttingAndDistribution(jobs chan protocol.Task) (int, *image.RGBA) {
	filePath := "images/input.jpg"
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Ошибка: не удалось найти файл по пути %s: %v\n", filePath, err)
		return 0, nil
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		fmt.Printf("Ошибка декодирования: %v\n", err)
		return 0, nil
	}
	fmt.Printf("Файл открыт. Формат: %s, Размер: %dx%d\n", format, img.Bounds().Dx(), img.Bounds().Dy())
	bounds := img.Bounds()
	rgbaImg := image.NewRGBA(bounds)
	draw.Draw(rgbaImg, bounds, img, bounds.Min, draw.Src)

	rowsPerTask := RowsPerTask
	stride := rgbaImg.Stride
	bytesPerTask := rowsPerTask * stride

	totalTasks := 0

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
		totalTasks++
	}

	return totalTasks, rgbaImg
}

func ImageCollector(results chan protocol.Result, rgbaImg *image.RGBA, totalTasks int) {
	completed := 0
	for res := range results {
		offset := res.Bounds.Min.Y * rgbaImg.Stride
		copy(rgbaImg.Pix[offset:], res.Payload)

		completed++
		fmt.Printf("Получен фрагмент %d. Прогресс: %d/%d\n", res.ID, completed, totalTasks)

		if completed == totalTasks {
			fmt.Println("Все части получены. Формируем файл...")

			saveImage(rgbaImg)
			fmt.Println("Работа завершена успешно!")
			os.Exit(0)
		}
	}
}

func saveImage(img *image.RGBA) {
	f, err := os.Create("images/output.png")
	if err != nil {
		fmt.Println("Ошибка при создании файла:", err)
		return
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		fmt.Println("Ошибка сохранения PNG:", err)
	}
}
