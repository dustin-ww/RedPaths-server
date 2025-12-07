package sse

import (
	"fmt"
	"gorm.io/gorm"
	"net/http"
)

func StartServer(port string, postgresCon *gorm.DB) {

	Init(postgresCon)

	http.HandleFunc("/sse", SSEHandler)
	http.HandleFunc("/trigger", TriggerEventHandler)

	fmt.Printf("SSE server running on :%s\n", port)
	http.ListenAndServe(":"+port, nil)
}
