package main

import (
	"dasa.cc/x/vk"
)

var appInfo = vk.AppInfo{
	Name: "Hello, Triangle",
	Version: vk.MakeVersion(1, 0, 0),
}

func main() {
	vk.Main(appInfo, func(app vk.Instance) {
		// Create Surface

		// Select GPU, creating logical device

		// Create swapchain

		// Run app loop
	})
}