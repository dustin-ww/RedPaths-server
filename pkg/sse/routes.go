package sse

import (
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

func StartServer(port string, postgresCon *gorm.DB) {

	Init(postgresCon)

	http.HandleFunc("/sse", SSEHandler)
	http.HandleFunc("/trigger", TriggerEventHandler)
	http.HandleFunc("/recommendation", RecommendationSSEHandler)

	fmt.Printf("SSE server running on :%s\n", port)
	http.ListenAndServe(":"+port, nil)
}
