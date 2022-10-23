package main

/*
#cgo pkg-config: vulkan glfw3

#define GLFW_INCLUDE_VULKAN
#include <GLFW/glfw3.h>
#include "main.h"
*/
import "C"

import (
	"fmt"
	"time"
	"unsafe"
)

const (
	width = 800
	height = 600
)

//export resizeCallback
func resizeCallback(window *C.GLFWwindow, width, height C.int) {
	fmt.Printf("resize event: width=%v height=%v\n", width, height)
}

//export errorCallback
func errorCallback(error int, description *C.char) {
	fmt.Printf("error %v: %s\n", error, C.GoString(description))
}

func main() {
	if C.glfwInit() == C.GLFW_FALSE {
		panic("glfwInit failed")
	}
	defer C.glfwTerminate()

	C.glfwSetErrorCallback((C.GLFWerrorfun)(unsafe.Pointer(C.errorCallback_cgo)))

	C.glfwWindowHint(C.GLFW_CLIENT_API, C.GLFW_NO_API)

	window := C.glfwCreateWindow(C.int(width), C.int(height), C.CString("Vulkan"), nil, nil)
	if window == nil {
		panic("glfwCreateWindow failed")
	}

	C.glfwSetFramebufferSizeCallback(window, (C.GLFWframebuffersizefun)(unsafe.Pointer(C.resizeCallback_cgo)))

	for C.glfwWindowShouldClose(window) == C.GLFW_FALSE {		
		fmt.Println("draw event", time.Now())
		C.glfwWaitEvents()
	}

	fmt.Printf("priority: %v\n", C.priority)

	fmt.Println("done")
}