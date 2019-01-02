package xapper

type Site struct {
	Devices [8]*Device
}

var Global *Site = &Site{}
