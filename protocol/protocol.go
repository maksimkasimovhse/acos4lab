package protocol

import "image"

type Task struct {
	ID      int             `json:"id"`
	Payload []byte          `json:"payload"`
	Bounds  image.Rectangle `json:"bounds"`
}

type Result struct {
	ID      int             `json:"id"`
	Payload []byte          `json:"payload"`
	Bounds  image.Rectangle `json:"bounds"`
}
