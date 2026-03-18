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
	"time"

	"acos/protocol"
)

const (
	RowsPerTask    = 100
	BufferCapacity = 100
)

func main() {
	jobs := make(chan protocol.Task, BufferCapacity)
	retry := make(chan protocol.Task, BufferCapacity)
	results := make(chan protocol.Result, BufferCapacity)
	next := make(chan bool, 1)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = ":8080"
	}

	ln, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Println("Ошибка: ", err)
	}
	fmt.Printf("Сервер слушает на порту: %s\n", port)

	for {
		var fileName string
		for {
			fmt.Println("Введите название файла(с расширением), который хотите инвертировать")
			_, err := fmt.Scan(&fileName)
			if err != nil {
				fmt.Println("Ошибка ввода: ", err)
				continue
			}

			if _, err := os.Stat("images/" + fileName); err != nil {
				fmt.Println("Файл не найден или недоступен")
				continue
			}

			break
		}

		totalTasks, rgbaImg := CuttingAndDistribution(jobs, fileName)
		if rgbaImg == nil {
			fmt.Println("Критическая ошибка: невозможно подготовить изображение")
			return
		}

		go ImageCollector(results, rgbaImg, totalTasks, next, fileName)

		fmt.Println("Ждем воркеров")
	WorkerLoop:
		for {
			ln.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
			conn, err := ln.Accept()

			select {
			case <-next:
				fmt.Println("Обработка окончена, можете переходить к следующему изображению")
				drainChannels(jobs, retry, results)
				break WorkerLoop
			default:

			}

			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				fmt.Println("Ошибка подключения к воркеру: ", err)
				continue
			}

			fmt.Println("Воркер подключился")
			go handleWorker(conn, jobs, retry, results)
		}

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
func CuttingAndDistribution(jobs chan protocol.Task, filename string) (int, *image.RGBA) {
	filePath := os.Getenv("INPUT_PATH")
	if filePath == "" {
		filePath = "images/"
	}

	file, err := os.Open(filePath + filename)
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

func ImageCollector(results chan protocol.Result, rgbaImg *image.RGBA, totalTasks int, next chan bool, fileName string) {
	completed := 0
	for res := range results {
		offset := res.Bounds.Min.Y * rgbaImg.Stride
		copy(rgbaImg.Pix[offset:], res.Payload)

		completed++
		fmt.Printf("Получен фрагмент %d. Прогресс: %d/%d\n", res.ID, completed, totalTasks)

		if completed == totalTasks {
			fmt.Println("Все части получены. Формируем файл...")

			saveImage(rgbaImg, fileName)
			fmt.Println("Работа завершена успешно!")
			next <- true
			return
		}
	}
}

func saveImage(img *image.RGBA, fileName string) {
	f, err := os.Create("images/" + fileName)
	if err != nil {
		fmt.Println("Ошибка при создании файла:", err)
		return
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		fmt.Println("Ошибка сохранения PNG:", err)
	}
}

func drainChannels(jobs, retry chan protocol.Task, results chan protocol.Result) {
	for len(jobs) > 0 {
		<-jobs
	}
	for len(retry) > 0 {
		<-retry
	}
	for len(results) > 0 {
		<-results
	}
}

//grafana для визуализации
