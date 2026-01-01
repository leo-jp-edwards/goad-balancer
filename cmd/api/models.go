package main

type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

type hostResponse struct {
	Host  string `json:"host"`
	Route string `json:"route"`
}
