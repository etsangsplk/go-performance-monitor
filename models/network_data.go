package models

import "time"

type NetworkData struct {
	InterfaceData []NetworkInterfaceData
	LastUpdate time.Time
}

type NetworkInterfaceData struct {
	Name string
	Rx float64
	Tx float64
}