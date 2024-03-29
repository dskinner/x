/*
https://vulkan-tutorial.com/en/Overview
https://github.com/toy80/vk/blob/master/toy80-example-vk/toy80-example-vk.go
*/
package vk

/*
#cgo pkg-config: vulkan glfw3

#define GLFW_INCLUDE_VULKAN
#include <GLFW/glfw3.h>
#include <stdlib.h>
#include "vk.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"math"
	"os"
	"unsafe"
)

//export resizeCallback
func resizeCallback(window *C.GLFWwindow, width, height C.int) {
	fmt.Printf("resize event: width=%v height=%v\n", width, height)
}

//export errorCallback
func errorCallback(error int, description *C.char) {
	fmt.Printf("error %v: %s\n", error, C.GoString(description))
}

var deviceExtensions = []string{
	"VK_KHR_swapchain",
}

func MakeVersion(major, minor, patch uint32) uint32 {
	return (major << 22) | (minor << 12) | patch
}

type AppInfo struct {
	Name    string
	Version uint32
}

type Instance struct {
	vkInstance C.VkInstance

	ApplicationName    string
	ApplicationVersion uint32

	EngineName    string
	EngineVersion uint32

	ValidationLayers       []string
	EnableValidationLayers bool

	debugMessenger C.VkDebugUtilsMessengerEXT
}

func requiredExtensions() (**C.char, C.uint) {
	var n C.uint
	p := C.glfwGetRequiredInstanceExtensions(&n)
	q := goStringSlice(p, n)
	q = append(q, "VK_EXT_debug_utils") // TODO somehow add  "vkGetInstanceProcAddr"
	return cStringArray(q)
}

var appInfo *C.VkApplicationInfo

func (instance *Instance) Create() error {
	// if instance.EnableValidationLayers && !checkValidationLayerSupport(instance.ValidationLayers) {
	// 	panic("enabled missing validation layer")
	// }

	appInfo = (*C.VkApplicationInfo)(C.malloc(C.sizeof_VkApplicationInfo))
	*appInfo = C.VkApplicationInfo{
		sType:              C.VK_STRUCTURE_TYPE_APPLICATION_INFO,
		pApplicationName:   C.CString(instance.ApplicationName),
		applicationVersion: C.uint(instance.ApplicationVersion),
		pEngineName:        C.CString(instance.EngineName),
		engineVersion:      C.uint(instance.EngineVersion),
		apiVersion:         C.VK_API_VERSION_1_0,
		pNext:              nil,
	}

	createInfo := C.VkInstanceCreateInfo{
		sType:            C.VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO,
		pApplicationInfo: appInfo,
	}
	createInfo.ppEnabledExtensionNames, createInfo.enabledExtensionCount = requiredExtensions()

	if instance.EnableValidationLayers && checkValidationLayerSupport(instance.ValidationLayers) {
		createInfo.ppEnabledLayerNames, createInfo.enabledLayerCount = cStringArray(instance.ValidationLayers)

		fmt.Println("setting up debug messenger for instance creation/destroy")

		debugCreateInfo := (*C.VkDebugUtilsMessengerCreateInfoEXT)(C.malloc(C.sizeof_VkDebugUtilsMessengerCreateInfoEXT))
		*debugCreateInfo = C.VkDebugUtilsMessengerCreateInfoEXT{
			sType:           C.VK_STRUCTURE_TYPE_DEBUG_UTILS_MESSENGER_CREATE_INFO_EXT,
			messageSeverity: C.VK_DEBUG_UTILS_MESSAGE_SEVERITY_VERBOSE_BIT_EXT | C.VK_DEBUG_UTILS_MESSAGE_SEVERITY_WARNING_BIT_EXT | C.VK_DEBUG_UTILS_MESSAGE_SEVERITY_ERROR_BIT_EXT,
			messageType:     C.VK_DEBUG_UTILS_MESSAGE_TYPE_GENERAL_BIT_EXT | C.VK_DEBUG_UTILS_MESSAGE_TYPE_VALIDATION_BIT_EXT | C.VK_DEBUG_UTILS_MESSAGE_TYPE_PERFORMANCE_BIT_EXT,
			pfnUserCallback: C.PFN_vkDebugUtilsMessengerCallbackEXT(C.debug_callback),
		}

		createInfo.pNext = unsafe.Pointer(debugCreateInfo)
	}

	//
	if C.vkCreateInstance(&createInfo, nil, &instance.vkInstance) != C.VK_SUCCESS {
		return fmt.Errorf("vkCreateInstance failed")
	}

	// TODO this needs to be moved to separate call for defer destroy on instance
	if instance.EnableValidationLayers && checkValidationLayerSupport(instance.ValidationLayers) {
		fmt.Println("creating debug messenger for everything else")

		debugCreateInfo := (*C.VkDebugUtilsMessengerCreateInfoEXT)(C.malloc(C.sizeof_VkDebugUtilsMessengerCreateInfoEXT))
		*debugCreateInfo = C.VkDebugUtilsMessengerCreateInfoEXT{
			sType:           C.VK_STRUCTURE_TYPE_DEBUG_UTILS_MESSENGER_CREATE_INFO_EXT,
			messageSeverity: C.VK_DEBUG_UTILS_MESSAGE_SEVERITY_VERBOSE_BIT_EXT | C.VK_DEBUG_UTILS_MESSAGE_SEVERITY_WARNING_BIT_EXT | C.VK_DEBUG_UTILS_MESSAGE_SEVERITY_ERROR_BIT_EXT,
			messageType:     C.VK_DEBUG_UTILS_MESSAGE_TYPE_GENERAL_BIT_EXT | C.VK_DEBUG_UTILS_MESSAGE_TYPE_VALIDATION_BIT_EXT | C.VK_DEBUG_UTILS_MESSAGE_TYPE_PERFORMANCE_BIT_EXT,
			pfnUserCallback: C.PFN_vkDebugUtilsMessengerCallbackEXT(C.debug_callback),
		}

		if C.CreateDebugUtilsMessengerEXT(instance.vkInstance, debugCreateInfo, nil, &instance.debugMessenger) != C.VK_SUCCESS {
			panic("CreateDebugUtilsMessengerEXT failed")
		}
	}

	return nil
}

func (instance *Instance) Destroy() {
	// TODO actually check if validation layers were enabled, not just the bool set by user
	C.DestroyDebugUtilsMessengerEXT(instance.vkInstance, instance.debugMessenger, nil)
	C.vkDestroyInstance(instance.vkInstance, nil)
}

func (instance *Instance) CreateWindowSurface(width, height int) (*WindowSurface, error) {
	surface := &WindowSurface{instance: instance}
	if err := surface.Create(width, height); err != nil {
		return nil, err
	}
	return surface, nil
}

type WindowSurface struct {
	instance  *Instance
	window    *C.GLFWwindow
	vkSurface C.VkSurfaceKHR
}

func (surface *WindowSurface) Create(width, height int) error {
	// init window
	C.glfwWindowHint(C.GLFW_CLIENT_API, C.GLFW_NO_API)
	// C.glfwWindowHint(C.GLFW_RESIZABLE, C.GLFW_FALSE)

	surface.window = C.glfwCreateWindow(C.int(width), C.int(height), C.CString("Vulkan"), nil, nil)
	if surface.window == nil {
		return errors.New("glfwCreateWindow failed")
	}

	C.glfwSetFramebufferSizeCallback(surface.window, (C.GLFWframebuffersizefun)(unsafe.Pointer(C.resizeCallback_cgo)))

	if C.glfwCreateWindowSurface(surface.instance.vkInstance, surface.window, nil, &surface.vkSurface) != C.VK_SUCCESS {
		return errors.New("failed to create window surface")
	}
	return nil
}

func (surface *WindowSurface) Destroy() {
	C.vkDestroySurfaceKHR(surface.instance.vkInstance, surface.vkSurface, nil)
	C.glfwDestroyWindow(surface.window)
}

func (surface *WindowSurface) EnumeratePhysicalDevices() C.VkPhysicalDevice {
	var n C.uint
	C.vkEnumeratePhysicalDevices(surface.instance.vkInstance, &n, nil)
	if n == 0 {
		panic("failed to find GPUs with Vulkan support")
	}
	p := (*C.VkPhysicalDevice)(C.malloc(C.size_t(n) * C.sizeof_VkPhysicalDevice))
	C.vkEnumeratePhysicalDevices(surface.instance.vkInstance, &n, p)
	devices := (*[1 << 30]C.VkPhysicalDevice)(unsafe.Pointer(p))[:n:n]

	fmt.Println("deviceCount:", n, len(devices))

	var physicalDevice C.VkPhysicalDevice
	for _, device := range devices {
		if isDeviceSuitable(device, surface.vkSurface) {
			physicalDevice = device
			break
		}
	}
	if physicalDevice == nil {
		panic("failed to find a suitable GPU")
	}

	return physicalDevice
}

func (surface *WindowSurface) CreateDevice() (Device, error) {
	// select default physical device
	// *** FIND PHYSICAL DEVICE ***
	// physical devices and queue families
	device := Device{surface: surface}
	device.vkPhysicalDevice = surface.EnumeratePhysicalDevices()
	device.getQueueFamilyIndices()

	// indices := GetQueueFamilyIndices(device.vkPhysicalDevice, surface.vkSurface)

	// ******************************************************************

	// *** MAKE LOGICAL DEVICE ***
	// https://vulkan-tutorial.com/en/Drawing_a_triangle/Setup/Logical_device_and_queues
	// logical device and queues

	queueCreateInfo := (*C.VkDeviceQueueCreateInfo)(C.malloc(C.sizeof_VkDeviceQueueCreateInfo))
	*queueCreateInfo = C.VkDeviceQueueCreateInfo{
		sType:            C.VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO,
		queueFamilyIndex: device.indices.graphicsFamily,
		queueCount:       1,
		pQueuePriorities: C.priority, // TODO find some other way, below fails
	}
	// queuePriority := (*C.float)(C.malloc(C.sizeof_float))
	// *queuePriority = 1
	// queueCreateInfo.pQueuePriorities = queuePriority

	deviceFeatures := (*C.VkPhysicalDeviceFeatures)(C.malloc(C.sizeof_VkPhysicalDeviceFeatures))
	*deviceFeatures = C.VkPhysicalDeviceFeatures{}
	// TODO enable features

	deviceCreateInfo := C.VkDeviceCreateInfo{
		sType:                C.VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO,
		pQueueCreateInfos:    queueCreateInfo,
		queueCreateInfoCount: 1,
		pEnabledFeatures:     deviceFeatures,
	}
	deviceCreateInfo.ppEnabledExtensionNames, deviceCreateInfo.enabledExtensionCount = cStringArray(deviceExtensions)

	// device validation layers should be ignored in up to date drivers
	if surface.instance.EnableValidationLayers {
		deviceCreateInfo.ppEnabledLayerNames, deviceCreateInfo.enabledLayerCount = cStringArray(surface.instance.ValidationLayers)
	}

	fmt.Println("creating logical device")
	if C.vkCreateDevice(device.vkPhysicalDevice, &deviceCreateInfo, nil, &device.vkDevice) != C.VK_SUCCESS {
		return Device{}, errors.New("failed to create logical device")
	}
	fmt.Println("logical device created")

	return device, nil
}

func checkInstanceExtensionSupport() {
	var n C.uint
	C.vkEnumerateInstanceExtensionProperties(nil, &n, nil)
	p := (*C.VkExtensionProperties)(C.malloc(C.size_t(n) * C.sizeof_VkExtensionProperties))
	C.vkEnumerateInstanceExtensionProperties(nil, &n, p)
	q := (*[1 << 30]C.VkExtensionProperties)(unsafe.Pointer(p))[:n:n]
	_ = q
	// TODO enforce constraints

	// for _, ext := range q {
	// 	fmt.Println("instance ext.name:", C.GoString(&ext.extensionName[0]))
	// }
}

func checkDeviceExtensionSupport(device C.VkPhysicalDevice) bool {
	var n C.uint
	C.vkEnumerateDeviceExtensionProperties(device, nil, &n, nil)
	p := (*C.VkExtensionProperties)(C.malloc(C.size_t(n) * C.sizeof_VkExtensionProperties))
	C.vkEnumerateDeviceExtensionProperties(device, nil, &n, p)
	q := (*[1 << 30]C.VkExtensionProperties)(unsafe.Pointer(p))[:n:n]
	_ = q

	// TODO enforce constraints

	// for _, ext := range q {
	// 	fmt.Println("extension:", C.GoString(&ext.extensionName[0]))
	// }
	// fmt.Println("len(ext):", n)

	return true
}

func checkValidationLayerSupport(validationLayers []string) bool {
	var n C.uint
	C.vkEnumerateInstanceLayerProperties(&n, nil)
	p := (*C.VkLayerProperties)(C.malloc(C.size_t(n) * C.sizeof_VkLayerProperties))
	C.vkEnumerateInstanceLayerProperties(&n, p)
	layers := (*[1 << 30]C.VkLayerProperties)(unsafe.Pointer(p))[:n:n]

	lookup := make(map[string]struct{})
	for _, layer := range layers {
		name := C.GoString(&layer.layerName[0])
		lookup[name] = struct{}{}
	}

	for _, layer := range validationLayers {
		if _, ok := lookup[layer]; !ok {
			fmt.Println("validation layer missing:", layer)
			return false
		}
	}

	return true
}

type Semaphore struct {
	vkSemaphore C.VkSemaphore
	vkDevice    C.VkDevice
}

func (s *Semaphore) Create() error {
	semaphoreInfo := C.VkSemaphoreCreateInfo{sType: C.VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO}
	if C.vkCreateSemaphore(s.vkDevice, &semaphoreInfo, nil, &s.vkSemaphore) != C.VK_SUCCESS {
		return errors.New("vkCreateSemaphore failed")
	}
	return nil
}

func (s Semaphore) Destroy() {
	C.vkDestroySemaphore(s.vkDevice, s.vkSemaphore, nil)
}

func Main(appInfo AppInfo, mainFn func(Instance)) {
	// init glfw
	if C.glfwInit() == C.GLFW_FALSE {
		panic("glfwInit failed")
	}
	defer C.glfwTerminate()

	C.glfwSetErrorCallback((C.GLFWerrorfun)(unsafe.Pointer(C.errorCallback_cgo)))

	// init vulkan
	instance := Instance{
		ApplicationName:    appInfo.Name,
		ApplicationVersion: appInfo.Version,
		EngineName:         "No Engine",
		EngineVersion:      MakeVersion(1, 0, 0),
		ValidationLayers: []string{
			"VK_LAYER_KHRONOS_validation",
		},
		EnableValidationLayers: true,
	}
	if err := instance.Create(); err != nil {
		panic(err)
	}
	defer instance.Destroy()

	const (
		width = 800
		height = 600
	)

	surface, err := instance.CreateWindowSurface(width, height)
	if err != nil {
		panic(err)
	}
	defer surface.Destroy()

	device, err := surface.CreateDevice()
	if err != nil {
		panic(err)
	}
	defer device.Destroy()

	// 0 for queue index, we only have one so hard-coded
	graphicsQueue := device.GetGraphicsQueue(0)
	presentQueue := device.GetPresentQueue(0)

	// create commandPool in prep for swapchain
	device.commandPool = device.CreateCommandPool(device.indices.graphicsFamily)
	defer C.vkDestroyCommandPool(device.vkDevice, device.commandPool, nil)

	// ******************************************************************
	// make swapchain
	swapchain := &Swapchain{device: device}
	if err := swapchain.Create(width, height); err != nil {
		panic(err)
	}
	// defer swapchain.Destroy()

	swapchain.CreateImageViews()
	swapchain.CreateRenderPass()
	swapchain.CreateGraphicsPipeline()
	swapchain.CreateFramebuffers()
	swapchain.CreateCommandBuffers()

	// ******************************************************************
	// *************************** Main Loop ****************************
	// ******************************************************************

	// https://vulkan-tutorial.com/en/Drawing_a_triangle/Drawing/Rendering_and_presentation

	// var imageAvailableSemaphore, renderFinishedSemaphore Semaphore
	// semaphoreInfo := C.VkSemaphoreCreateInfo{sType: C.VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO}
	// if C.vkCreateSemaphore(device.vkDevice, &semaphoreInfo, nil, &imageAvailableSemaphore) != C.VK_SUCCESS {
	// 	panic("vkCreateSemaphore failed")
	// }
	// defer C.vkDestroySemaphore(device.vkDevice, imageAvailableSemaphore, nil)
	// if C.vkCreateSemaphore(device.vkDevice, &semaphoreInfo, nil, &renderFinishedSemaphore) != C.VK_SUCCESS {
	// 	panic("vkCreateSemaphore failed")
	// }
	// defer C.vkDestroySemaphore(device.vkDevice, renderFinishedSemaphore, nil)

	imageAvailableSemaphore := device.CreateSemaphore()
	defer imageAvailableSemaphore.Destroy()

	renderFinishedSemaphore := device.CreateSemaphore()
	defer renderFinishedSemaphore.Destroy()

	// draw frame

	// allocate once outside of loop since im being bad with memory right now

	// VkSemaphore waitSemaphores[] = {imageAvailableSemaphore};
	pWaitSemaphores := (*C.VkSemaphore)(C.malloc(C.sizeof_VkSemaphore))
	qWaitSemaphores := (*[1 << 30]C.VkSemaphore)(unsafe.Pointer(pWaitSemaphores))[:1:1]
	qWaitSemaphores[0] = imageAvailableSemaphore.vkSemaphore

	// VkPipelineStageFlags waitStages[] = {VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT};
	pWaitStages := (*C.VkPipelineStageFlags)(C.malloc(C.sizeof_VkPipelineStageFlags))
	qWaitStages := (*[1 << 30]C.VkPipelineStageFlags)(unsafe.Pointer(pWaitStages))[:1:1]
	qWaitStages[0] = C.VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT

	// VkSemaphore signalSemaphores[] = {renderFinishedSemaphore};
	pSignalSemaphores := (*C.VkSemaphore)(C.malloc(C.sizeof_VkSemaphore))
	qSignalSemaphores := (*[1 << 30]C.VkSemaphore)(unsafe.Pointer(pSignalSemaphores))[:1:1]
	qSignalSemaphores[0] = renderFinishedSemaphore.vkSemaphore

	// VkSwapchainKHR swapchains[] = {swapchain};
	pSwapchains := (*C.VkSwapchainKHR)(C.malloc(C.sizeof_VkSwapchainKHR))
	qSwapchains := (*[1 << 30]C.VkSwapchainKHR)(unsafe.Pointer(pSwapchains))[:1:1]
	qSwapchains[0] = swapchain.vkSwapchain

	// main loop
	for C.glfwWindowShouldClose(surface.window) == C.GLFW_FALSE {
		C.glfwPollEvents()
		// C.glfwWaitEvents()

		// draw frame
		var imageIndex C.uint // references the index in our swapchain.Images that's available
		result := C.vkAcquireNextImageKHR(device.vkDevice, swapchain.vkSwapchain, math.MaxUint64, imageAvailableSemaphore.vkSemaphore, C.VkFence(vkNullHandle), &imageIndex)
		if result == C.VK_ERROR_OUT_OF_DATE_KHR {
			// recreate swapchain
			fmt.Println("swapchain out of date")

			C.vkDeviceWaitIdle(device.vkDevice)
			swapchain.Destroy()

			// swapchain = &Swapchain{device: device}
			if err := swapchain.Create(width, height); err != nil {
				panic(err)
			}
			qSwapchains[0] = swapchain.vkSwapchain

			fmt.Println("swapchain.CreateImageViews()")
			swapchain.CreateImageViews()
			fmt.Println("swapchain.CreateRenderPass()")
			swapchain.CreateRenderPass()
			fmt.Println("swapchain.CreateGraphicsPipeline()")
			swapchain.CreateGraphicsPipeline()
			fmt.Println("swapchain.CreateFramebuffers()")
			swapchain.CreateFramebuffers()

			// not properly freeing in swapchain.Destroy
			swapchain.CreateCommandBuffers()

			continue
		}

		// VkSemaphore waitSemaphores[] = {imageAvailableSemaphore};
		// VkPipelineStageFlags waitStages[] = {VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT};
		// VkSemaphore signalSemaphores[] = {renderFinishedSemaphore};

		submitInfo := (*C.VkSubmitInfo)(C.malloc(C.sizeof_VkSubmitInfo))
		*submitInfo = C.VkSubmitInfo{
			sType:                C.VK_STRUCTURE_TYPE_SUBMIT_INFO,
			waitSemaphoreCount:   1,
			pWaitSemaphores:      pWaitSemaphores,
			pWaitDstStageMask:    pWaitStages,
			commandBufferCount:   1,
			pCommandBuffers:      &swapchain.commandBuffers[imageIndex],
			signalSemaphoreCount: 1,
			pSignalSemaphores:    pSignalSemaphores,
		}

		if C.vkQueueSubmit(graphicsQueue, 1, submitInfo, C.VkFence(vkNullHandle)) != C.VK_SUCCESS {
			panic("vkQueueSubmit")
		}

		presentInfo := (*C.VkPresentInfoKHR)(C.malloc(C.sizeof_VkPresentInfoKHR))
		*presentInfo = C.VkPresentInfoKHR{
			sType:              C.VK_STRUCTURE_TYPE_PRESENT_INFO_KHR,
			waitSemaphoreCount: 1,
			pWaitSemaphores:    pSignalSemaphores,
			swapchainCount:     1,
			pSwapchains:        pSwapchains,
			pImageIndices:      &imageIndex,
		}

		if res := C.vkQueuePresentKHR(presentQueue, presentInfo); res == C.VK_ERROR_OUT_OF_DATE_KHR {
			fmt.Println("queue present failed, swapchain out of date")
		}

		//
		C.vkQueueWaitIdle(presentQueue)

		//
		C.free(unsafe.Pointer(submitInfo))
		C.free(unsafe.Pointer(presentInfo))
	}

	C.vkDeviceWaitIdle(device.vkDevice)
	swapchain.Destroy()
}

type CommandBuffer struct {
	vkCommandBuffer C.VkCommandBuffer
}

func (cmd CommandBuffer) BeginCommandBuffer() {

}

func (cmd CommandBuffer) BeginRenderPass() {}

func (cmd CommandBuffer) BindPipeline() {}

func (cmd CommandBuffer) Draw() {}

func (cmd CommandBuffer) EndRenderPass() {}

func (cmd CommandBuffer) EndCommandBuffer() {}

func (dev Device) CreateCommandPool(queueFamilyIndex C.uint) C.VkCommandPool {
	poolInfo := C.VkCommandPoolCreateInfo{
		sType:            C.VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
		queueFamilyIndex: queueFamilyIndex,
		flags:            0, // Optional
	}
	var commandPool C.VkCommandPool
	if C.vkCreateCommandPool(dev.vkDevice, &poolInfo, nil, &commandPool) != C.VK_SUCCESS {
		panic("vkCreateCommandPool failed")
	}
	return commandPool
}

type ShaderModule struct {
	vkShaderModule C.VkShaderModule
}

func (sm ShaderModule) Destroy(dev Device) {
	C.vkDestroyShaderModule(dev.vkDevice, sm.vkShaderModule, nil)
}

type ShaderAsset string

func (name ShaderAsset) Module(dev Device) ShaderModule {
	return dev.CreateShaderModule(mustReadFile(string(name)))
}

func (dev Device) CreateShaderModule(code []byte) ShaderModule {
	// TODO this may not be appropriate, passing in bytes allocated in go to c,
	// but i think it should be ok, just double check at some point
	createInfo := (*C.VkShaderModuleCreateInfo)(C.malloc(C.sizeof_VkShaderModuleCreateInfo))
	*createInfo = C.VkShaderModuleCreateInfo{
		sType:    C.VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
		codeSize: C.size_t(len(code)),
		pCode:    (*C.uint)(unsafe.Pointer(&code[0])),
	}

	var shaderModule ShaderModule
	if C.vkCreateShaderModule(dev.vkDevice, createInfo, nil, &shaderModule.vkShaderModule) != C.VK_SUCCESS {
		panic("vkCreateShaderModule failed")
	}
	return shaderModule
}

type Pipeline struct {
	vkDevice C.VkDevice
	vkPipeline C.VkPipeline
	vkPipelineLayout C.VkPipelineLayout
}

func (pipeline Pipeline) Destroy() {
	fmt.Printf("Pipeline: %+v\n", pipeline)
	C.vkDestroyPipeline(pipeline.vkDevice, pipeline.vkPipeline, nil)
	C.vkDestroyPipelineLayout(pipeline.vkDevice, pipeline.vkPipelineLayout, nil)
}

func (dev Device) CreateGraphicsPipeline(swapchain *Swapchain, renderPass C.VkRenderPass) Pipeline {
	pipeline := Pipeline{vkDevice: dev.vkDevice}

	vertAsset := ShaderAsset("shaders/vert.spv")
	fragAsset := ShaderAsset("shaders/frag.spv")

	// vertShaderCode := mustReadFile("shaders/vert.spv")
	// fragShaderCode := mustReadFile("shaders/frag.spv")

	// vertShaderModule := createShaderModule(device, vertShaderCode)
	// defer C.vkDestroyShaderModule(device, vertShaderModule, nil)
	// fragShaderModule := createShaderModule(device, fragShaderCode)
	// defer C.vkDestroyShaderModule(device, fragShaderModule, nil)

	vertModule := vertAsset.Module(dev)
	fragModule := fragAsset.Module(dev)

	vertShaderStageInfo := C.VkPipelineShaderStageCreateInfo{
		sType:  C.VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
		stage:  C.VK_SHADER_STAGE_VERTEX_BIT,
		module: vertModule.vkShaderModule,
		pName:  C.CString("main"),
	}
	fragShaderStageInfo := C.VkPipelineShaderStageCreateInfo{
		sType:  C.VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
		stage:  C.VK_SHADER_STAGE_FRAGMENT_BIT,
		module: fragModule.vkShaderModule,
		pName:  C.CString("main"),
	}
	shaderStages := []C.VkPipelineShaderStageCreateInfo{
		vertShaderStageInfo,
		fragShaderStageInfo,
	}
	pShaderStages := (*C.VkPipelineShaderStageCreateInfo)(C.malloc(C.size_t(len(shaderStages)) * C.sizeof_VkPipelineShaderStageCreateInfo))
	qShaderStages := (*[1 << 30]C.VkPipelineShaderStageCreateInfo)(unsafe.Pointer(pShaderStages))[:len(shaderStages):len(shaderStages)]
	for i := range qShaderStages {
		qShaderStages[i] = shaderStages[i]
	}

	//
	// https://vulkan-tutorial.com/en/Drawing_a_triangle/Graphics_pipeline_basics/Fixed_functions
	//

	vertexInputInfo := (*C.VkPipelineVertexInputStateCreateInfo)(C.malloc(C.sizeof_VkPipelineVertexInputStateCreateInfo))
	*vertexInputInfo = C.VkPipelineVertexInputStateCreateInfo{
		sType:                           C.VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO,
		vertexBindingDescriptionCount:   0,
		pVertexBindingDescriptions:      nil, // Optional
		vertexAttributeDescriptionCount: 0,
		pVertexAttributeDescriptions:    nil, // Optional
	}

	inputAssembly := (*C.VkPipelineInputAssemblyStateCreateInfo)(C.malloc(C.sizeof_VkPipelineInputAssemblyStateCreateInfo))
	*inputAssembly = C.VkPipelineInputAssemblyStateCreateInfo{
		sType:                  C.VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO,
		topology:               C.VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
		primitiveRestartEnable: C.VK_FALSE,
	}

	viewport := (*C.VkViewport)(C.malloc(C.sizeof_VkViewport))
	*viewport = C.VkViewport{
		x:        0.0,
		y:        0.0,
		width:    C.float(swapchain.Extent.width),
		height:   C.float(swapchain.Extent.height),
		minDepth: 0.0,
		maxDepth: 1.0,
	}

	scissor := (*C.VkRect2D)(C.malloc(C.sizeof_VkRect2D))
	*scissor = C.VkRect2D{
		offset: C.VkOffset2D{0, 0},
		extent: swapchain.Extent,
	}

	viewportState := (*C.VkPipelineViewportStateCreateInfo)(C.malloc(C.sizeof_VkPipelineViewportStateCreateInfo))
	*viewportState = C.VkPipelineViewportStateCreateInfo{
		sType:         C.VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO,
		viewportCount: 1,
		pViewports:    viewport,
		scissorCount:  1,
		pScissors:     scissor,
	}

	rasterizer := (*C.VkPipelineRasterizationStateCreateInfo)(C.malloc(C.sizeof_VkPipelineRasterizationStateCreateInfo))
	*rasterizer = C.VkPipelineRasterizationStateCreateInfo{
		sType:                   C.VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO,
		depthClampEnable:        C.VK_FALSE,
		rasterizerDiscardEnable: C.VK_FALSE,
		polygonMode:             C.VK_POLYGON_MODE_FILL,
		lineWidth:               1.0,
		cullMode:                C.VK_CULL_MODE_BACK_BIT,
		frontFace:               C.VK_FRONT_FACE_CLOCKWISE,
		depthBiasEnable:         C.VK_FALSE,
		depthBiasConstantFactor: 0.0, // Optional
		depthBiasClamp:          0.0, // Optional
		depthBiasSlopeFactor:    0.0, // Optional
	}

	multisampling := (*C.VkPipelineMultisampleStateCreateInfo)(C.malloc(C.sizeof_VkPipelineMultisampleStateCreateInfo))
	*multisampling = C.VkPipelineMultisampleStateCreateInfo{
		sType:                 C.VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO,
		sampleShadingEnable:   C.VK_FALSE,
		rasterizationSamples:  C.VK_SAMPLE_COUNT_1_BIT,
		minSampleShading:      1.0,        // Optional
		pSampleMask:           nil,        // Optional
		alphaToCoverageEnable: C.VK_FALSE, // Optional
		alphaToOneEnable:      C.VK_FALSE, // Optional
	}

	// check tutorial for changes in alpha blending required for this and other structs

	colorBlendAttachment := (*C.VkPipelineColorBlendAttachmentState)(C.malloc(C.sizeof_VkPipelineColorBlendAttachmentState))
	*colorBlendAttachment = C.VkPipelineColorBlendAttachmentState{
		colorWriteMask:      C.VK_COLOR_COMPONENT_R_BIT | C.VK_COLOR_COMPONENT_G_BIT | C.VK_COLOR_COMPONENT_B_BIT | C.VK_COLOR_COMPONENT_A_BIT,
		blendEnable:         C.VK_FALSE,
		srcColorBlendFactor: C.VK_BLEND_FACTOR_ONE,  // Optional
		dstColorBlendFactor: C.VK_BLEND_FACTOR_ZERO, // Optional
		colorBlendOp:        C.VK_BLEND_OP_ADD,      // Optional
		srcAlphaBlendFactor: C.VK_BLEND_FACTOR_ONE,  // Optional
		dstAlphaBlendFactor: C.VK_BLEND_FACTOR_ZERO, // Optional
		alphaBlendOp:        C.VK_BLEND_OP_ADD,      // Optional
	}

	colorBlending := (*C.VkPipelineColorBlendStateCreateInfo)(C.malloc(C.sizeof_VkPipelineColorBlendStateCreateInfo))
	*colorBlending = C.VkPipelineColorBlendStateCreateInfo{
		sType:           C.VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO,
		logicOpEnable:   C.VK_FALSE,
		logicOp:         C.VK_LOGIC_OP_COPY, // Optional
		attachmentCount: 1,
		pAttachments:    colorBlendAttachment,
	}
	colorBlending.blendConstants[0] = 0.0 // Optional
	colorBlending.blendConstants[1] = 0.0 // Optional
	colorBlending.blendConstants[2] = 0.0 // Optional
	colorBlending.blendConstants[3] = 0.0 // Optional

	// TODO allocate in C
	dynamicStates := []C.VkDynamicState{
		C.VK_DYNAMIC_STATE_VIEWPORT,
		C.VK_DYNAMIC_STATE_LINE_WIDTH,
	}
	pDynamicStates := (*C.VkDynamicState)(C.malloc(C.size_t(len(dynamicStates)) * C.sizeof_VkDynamicState))
	qDynamicStates := (*[1 << 30]C.VkDynamicState)(unsafe.Pointer(pDynamicStates))[:len(dynamicStates):len(dynamicStates)]
	for i := range qDynamicStates {
		qDynamicStates[i] = dynamicStates[i]
	}

	dynamicState := (*C.VkPipelineDynamicStateCreateInfo)(C.malloc(C.sizeof_VkPipelineDynamicStateCreateInfo))
	*dynamicState = C.VkPipelineDynamicStateCreateInfo{
		sType:             C.VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO,
		dynamicStateCount: 2,
		pDynamicStates:    pDynamicStates,
	}

	pipelineLayoutInfo := C.VkPipelineLayoutCreateInfo{
		sType:                  C.VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
		setLayoutCount:         0,   // Optional
		pSetLayouts:            nil, // Optional
		pushConstantRangeCount: 0,   // Optional
		pPushConstantRanges:    nil, // Optional
	}
	if C.vkCreatePipelineLayout(dev.vkDevice, &pipelineLayoutInfo, nil, &pipeline.vkPipelineLayout) != C.VK_SUCCESS {
		panic("vkCreatePipelineLayout failed")
	}

	pipelineInfo := C.VkGraphicsPipelineCreateInfo{
		sType:      C.VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO,
		stageCount: 2,
		pStages:    pShaderStages,

		pVertexInputState:   vertexInputInfo,
		pInputAssemblyState: inputAssembly,
		pViewportState:      viewportState,
		pRasterizationState: rasterizer,
		pMultisampleState:   multisampling,
		pDepthStencilState:  nil, // Optional
		pColorBlendState:    colorBlending,
		pDynamicState:       nil, // Optional

		layout: pipeline.vkPipelineLayout,

		renderPass: renderPass,
		subpass:    0,

		// basePipelineHandle: VK_NULL_HANDLE, // Optional
		// basePipelineIndex:  -1,             // Optional
	}
	if C.vkCreateGraphicsPipelines(dev.vkDevice, C.VkPipelineCache(vkNullHandle), 1, &pipelineInfo, nil, &pipeline.vkPipeline) != C.VK_SUCCESS {
		panic("vkCreateGraphicsPipelines failed")
	}

	return pipeline
}

func isDeviceSuitable(device C.VkPhysicalDevice, surface C.VkSurfaceKHR) bool {
	var (
		properties C.VkPhysicalDeviceProperties
		features   C.VkPhysicalDeviceFeatures
	)

	C.vkGetPhysicalDeviceProperties(device, &properties)
	C.vkGetPhysicalDeviceFeatures(device, &features)

	// TODO check queue family is ok for given device
	checkDeviceExtensionSupport(device)

	swapchainOk := false
	// if extensionsSupported .. not really relevant to me
	swapChainSupport := GetSwapchainSupport(device, surface)
	swapchainOk = swapChainSupport.Formats != nil && swapChainSupport.PresentModes != nil

	return properties.deviceType == C.VK_PHYSICAL_DEVICE_TYPE_DISCRETE_GPU && swapchainOk // && features.geometryShader == 1
	/* From tutorial
	It is important that we only try to query for swap chain support after verifying that the extension is available. The last line of the function changes to:
	return indices.isComplete() && extensionsSupported && swapChainAdequate;
	*/
}

// *******************
// Device
// *******************

type Queue struct {
	vkQueue C.VkQueue
}

// Submit a sequence of semaphores or command buffers to the queue.
func (queue Queue) Submit(infos ...C.VkSubmitInfo) {

}

// Device represents a logical device tied to a surface.
type Device struct {
	vkPhysicalDevice C.VkPhysicalDevice
	vkDevice         C.VkDevice

	surface *WindowSurface

	indices QueueFamilyIndices

	commandPool C.VkCommandPool
}

type QueueFamilyIndices struct {
	graphicsFamily   C.uint
	graphicsFamilyOk bool

	presentFamily   C.uint
	presentFamilyOk bool
}

func (device *Device) getQueueFamilyIndices() {
	var n C.uint
	C.vkGetPhysicalDeviceQueueFamilyProperties(device.vkPhysicalDevice, &n, nil)
	p := (*C.VkQueueFamilyProperties)(C.malloc(C.size_t(n) * C.sizeof_VkQueueFamilyProperties))
	C.vkGetPhysicalDeviceQueueFamilyProperties(device.vkPhysicalDevice, &n, p)
	queueFamilies := (*[1 << 30]C.VkQueueFamilyProperties)(unsafe.Pointer(p))[:n:n]

	fmt.Println("n queueFamilies:", len(queueFamilies))

	for i, queueFamily := range queueFamilies {
		fmt.Printf("%v: %b\n", i, queueFamily.queueFlags)
		if queueFamily.queueFlags&C.VK_QUEUE_GRAPHICS_BIT == 1 {
			fmt.Println("found graphics queue family:", i)
			device.indices.graphicsFamily = C.uint(i)
			device.indices.graphicsFamilyOk = true
		}

		// TODO more than likely this is the same queue as above, and likely want to prefer
		// device queue that supports only both for improved performance.
		var presentSupport C.VkBool32 = 0
		C.vkGetPhysicalDeviceSurfaceSupportKHR(device.vkPhysicalDevice, C.uint(i), device.surface.vkSurface, &presentSupport)
		if presentSupport == 1 {
			fmt.Println("found present family:", i)
			device.indices.presentFamily = C.uint(i)
			device.indices.presentFamilyOk = true
		}

		if device.indices.graphicsFamilyOk && device.indices.presentFamilyOk {
			break
		}
	}
}

func (device Device) Destroy() {
	C.vkDestroyDevice(device.vkDevice, nil)
}

func (device Device) GetQueue(familyIndex C.uint, index C.uint) C.VkQueue {
	var queue C.VkQueue
	C.vkGetDeviceQueue(device.vkDevice, familyIndex, index, &queue)
	return queue
}

func (device Device) GetGraphicsQueue(index C.uint) C.VkQueue {
	return device.GetQueue(device.indices.graphicsFamily, index)
}

func (device Device) GetPresentQueue(index C.uint) C.VkQueue {
	return device.GetQueue(device.indices.presentFamily, index)
}

// func (device Device) CreateSwapchain(width, height uint) (Swapchain, error) {
// 	swapchain := Swapchain{
// 		device: device,
// 	}
// 	if err := swapchain.Create(device, width, height); err != nil {
// 		return Swapchain{}, err
// 	}
// 	return swapchain, nil
// }

func (device Device) CreateSemaphore() Semaphore {
	s := Semaphore{vkDevice: device.vkDevice}
	if err := s.Create(); err != nil {
		panic(err)
	}
	return s
}

func (dev Device) CreateRenderPass(swapchain *Swapchain) C.VkRenderPass {
	colorAttachment := (*C.VkAttachmentDescription)(C.malloc(C.sizeof_VkAttachmentDescription))
	*colorAttachment = C.VkAttachmentDescription{
		format:         swapchain.ImageFormat,
		samples:        C.VK_SAMPLE_COUNT_1_BIT,
		loadOp:         C.VK_ATTACHMENT_LOAD_OP_CLEAR,
		storeOp:        C.VK_ATTACHMENT_STORE_OP_STORE,
		stencilLoadOp:  C.VK_ATTACHMENT_LOAD_OP_DONT_CARE,
		stencilStoreOp: C.VK_ATTACHMENT_STORE_OP_DONT_CARE,
		initialLayout:  C.VK_IMAGE_LAYOUT_UNDEFINED,
		finalLayout:    C.VK_IMAGE_LAYOUT_PRESENT_SRC_KHR,
	}

	colorAttachmentRef := (*C.VkAttachmentReference)(C.malloc(C.sizeof_VkAttachmentReference))
	*colorAttachmentRef = C.VkAttachmentReference{
		attachment: 0,
		layout:     C.VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
	}

	subpass := (*C.VkSubpassDescription)(C.malloc(C.sizeof_VkSubpassDescription))
	*subpass = C.VkSubpassDescription{
		pipelineBindPoint:    C.VK_PIPELINE_BIND_POINT_GRAPHICS,
		colorAttachmentCount: 1,
		pColorAttachments:    colorAttachmentRef,
	}

	dependency := (*C.VkSubpassDependency)(C.malloc(C.sizeof_VkSubpassDependency))
	*dependency = C.VkSubpassDependency{
		srcSubpass:    C.VK_SUBPASS_EXTERNAL,
		dstSubpass:    0,
		srcStageMask:  C.VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
		srcAccessMask: 0,
		dstStageMask:  C.VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
		dstAccessMask: C.VK_ACCESS_COLOR_ATTACHMENT_READ_BIT | C.VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT,
	}

	var renderPass C.VkRenderPass
	renderPassInfo := C.VkRenderPassCreateInfo{
		sType:           C.VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO,
		attachmentCount: 1,
		pAttachments:    colorAttachment,
		subpassCount:    1,
		pSubpasses:      subpass,
		dependencyCount: 1,
		pDependencies:   dependency,
	}
	if C.vkCreateRenderPass(dev.vkDevice, &renderPassInfo, nil, &renderPass) != C.VK_SUCCESS {
		panic("vkCreateRenderPass failed")
	}
	return renderPass
}

// *******************
// Utils
// *******************

var vkNullHandle = unsafe.Pointer(uintptr(0))

func cStringArray(xs []string) (**C.char, C.uint) {
	p := C.malloc(C.size_t(len(xs)) * C.size_t(unsafe.Sizeof(uintptr(0))))
	q := (*[1 << 30]*C.char)(p)
	for i, x := range xs {
		q[i] = C.CString(x)
	}
	return (**C.char)(p), C.uint(len(xs))
}

func goStringSlice(p **C.char, n C.uint) []string {
	xs := make([]string, int(n))
	q := (*[1 << 30]*C.char)(unsafe.Pointer(p))[:int(n):int(n)]
	for i, x := range q {
		xs[i] = C.GoString(x)
	}
	return xs
}

func mustReadFile(filename string) []byte {
	bin, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return bin
}

func cclampu32(a, min, max C.uint) C.uint {
	return cmaxu32(min, cminu32(max, a))
}

func cmaxu32(a, b C.uint) C.uint {
	if a > b {
		return a
	}
	return b
}

func cminu32(a, b C.uint) C.uint {
	if a < b {
		return a
	}
	return b
}
