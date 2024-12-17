package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (T *Tasks) getStatus(c *gin.Context) {

	T.statusLock.Lock()
	jsonData, err := json.Marshal(T.RunStatus)
	fmt.Println(T.RunStatus)
	fmt.Println(string(jsonData))
	if err != nil {
		c.String(http.StatusServiceUnavailable, err.Error())

	}

	c.String(http.StatusOK, string(jsonData))
	T.statusLock.Unlock()

}
