#define GLFW_INCLUDE_VULKAN
#include <GLFW/glfw3.h>
#include "main.h"
#include "_cgo_export.h"

float _priority = 1.0f;
float* priority = &_priority;

void errorCallback_cgo(int error, char* description) {
	errorCallback(error, description);
}

void resizeCallback_cgo(GLFWwindow* window, int width, int height) {
	resizeCallback(window, width, height);
}