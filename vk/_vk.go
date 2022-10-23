/*
https://vulkan-tutorial.com/en/Overview
https://github.com/toy80/vk/blob/master/toy80-example-vk/toy80-example-vk.go
*/
//go:generate glslc shaders/shader.vert -o shaders/vert.spv
//go:generate glslc shaders/shader.frag -o shaders/frag.spv
package vk

/*
#cgo pkg-config: vulkan glfw3

//#define VK_USE_PLATFORM_WIN32_KHR
//#define GLFW_EXPOSE_NATIVE_WIN32

//#define VK_USE_PLATFORM_XCB_KHR
//#define GLFW_EXPOSE_NATIVE_X11

#define GLFW_INCLUDE_VULKAN
#include <GLFW/glfw3.h>

//#include <GLFW/glfw3native.h>

#include <stdio.h>
#include <stdlib.h>

float _priority = 1.0f;
float* priority = &_priority;

VkClearValue _defaultClearColor = {0.0, 0.0, 0.0, 1.0};
VkClearValue* defaultClearColor = &_defaultClearColor;

void error_callback(int error, const char* description) {
	fprintf(stderr, "Error: %s\n", description);
}

void init_error_callback() {
	glfwSetErrorCallback(error_callback);
}

// vulkan layer debug messenger callback
VkBool32 debug_callback(
    VkDebugUtilsMessageSeverityFlagBitsEXT           messageSeverity,
    VkDebugUtilsMessageTypeFlagsEXT                  messageTypes,
    const VkDebugUtilsMessengerCallbackDataEXT*      pCallbackData,
    void*                                            pUserData
) {
	if (messageSeverity >= VK_DEBUG_UTILS_MESSAGE_SEVERITY_WARNING_BIT_EXT) {
		fprintf(stderr, "validation layer: %s\n", pCallbackData->pMessage);
	}
	return VK_FALSE;
}

VkResult CreateDebugUtilsMessengerEXT(VkInstance instance, const VkDebugUtilsMessengerCreateInfoEXT* pCreateInfo, const VkAllocationCallbacks* pAllocator, VkDebugUtilsMessengerEXT* pDebugMessenger) {
    PFN_vkCreateDebugUtilsMessengerEXT func = (PFN_vkCreateDebugUtilsMessengerEXT) vkGetInstanceProcAddr(instance, "vkCreateDebugUtilsMessengerEXT");
    if (func != NULL) {
        return func(instance, pCreateInfo, pAllocator, pDebugMessenger);
    } else {
        return VK_ERROR_EXTENSION_NOT_PRESENT;
    }
}

void DestroyDebugUtilsMessengerEXT(VkInstance instance, VkDebugUtilsMessengerEXT debugMessenger, const VkAllocationCallbacks* pAllocator) {
    PFN_vkDestroyDebugUtilsMessengerEXT func = (PFN_vkDestroyDebugUtilsMessengerEXT) vkGetInstanceProcAddr(instance, "vkDestroyDebugUtilsMessengerEXT");
    if (func != NULL) {
        func(instance, debugMessenger, pAllocator);
    }
}

*/
import "C"

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"unsafe"
)

var deviceExtensions = []string{
	"VK_KHR_swapchain",
}

const (
	Width  = 800
	Height = 600
)

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

func (instance *Instance) CreateWindowSurface() (*WindowSurface, error) {
	surface := &WindowSurface{instance: instance}
	if err := surface.Create(); err != nil {
		return nil, err
	}
	return surface, nil
}

type WindowSurface struct {
	instance  *Instance
	window    *C.GLFWwindow
	vkSurface C.VkSurfaceKHR
}

func (surface *WindowSurface) Create() error {
	// init window
	C.glfwWindowHint(C.GLFW_CLIENT_API, C.GLFW_NO_API)
	C.glfwWindowHint(C.GLFW_RESIZABLE, C.GLFW_FALSE)

	surface.window = C.glfwCreateWindow(Width, Height, C.CString("Vulkan"), nil, nil)
	if surface.window == nil {
		return errors.New("glfwCreateWindow failed")
	}

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
	C.init_error_callback()

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

	surface, err := instance.CreateWindowSurface()
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

	// ******************************************************************
	// make swapchain
	swapchain, err := device.CreateSwapchain()
	if err != nil {
		panic(err)
	}
	defer swapchain.Destroy()

	swapchain.CreateImageViews()
	defer swapchain.DestroyImageViews()

	// ******************************************************************
	// *********************** GRAPHICS PIPELINE ************************
	// ******************************************************************

	renderPass := createRenderPass(device.vkDevice, swapchain)
	defer C.vkDestroyRenderPass(device.vkDevice, renderPass, nil)

	graphicsPipeline, pipelineLayout := createGraphicsPipeline(device, swapchain, renderPass)
	defer C.vkDestroyPipelineLayout(device.vkDevice, pipelineLayout, nil)
	defer C.vkDestroyPipeline(device.vkDevice, graphicsPipeline, nil)

	// ******************************************************************
	// ************************** Framebuffers **************************
	// ******************************************************************

	framebuffers := createFramebuffers(device.vkDevice, swapchain, renderPass)
	defer func() {
		for _, framebuffer := range framebuffers {
			C.vkDestroyFramebuffer(device.vkDevice, framebuffer, nil)
		}
	}()

	// ******************************************************************
	// ******************************************************************
	// ******************************************************************

	commandPool := createCommandPool(device.vkDevice, device.indices)
	defer C.vkDestroyCommandPool(device.vkDevice, commandPool, nil)

	commandBuffers := createCommandBuffers(device.vkDevice, commandPool, framebuffers, renderPass, swapchain, graphicsPipeline)

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
		C.vkAcquireNextImageKHR(device.vkDevice, swapchain.vkSwapchain, math.MaxUint64, imageAvailableSemaphore.vkSemaphore, C.VkFence(vkNullHandle), &imageIndex)

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
			pCommandBuffers:      &commandBuffers[imageIndex],
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

		C.vkQueuePresentKHR(presentQueue, presentInfo)

		//
		C.vkQueueWaitIdle(presentQueue)

		//
		C.free(unsafe.Pointer(submitInfo))
		C.free(unsafe.Pointer(presentInfo))
	}

	C.vkDeviceWaitIdle(device.vkDevice)
}

func createCommandBuffers(device C.VkDevice, commandPool C.VkCommandPool, framebuffers []C.VkFramebuffer, renderPass C.VkRenderPass, swapchain Swapchain, graphicsPipeline C.VkPipeline) []C.VkCommandBuffer {
	commandBuffers := make([]C.VkCommandBuffer, len(framebuffers))

	allocInfo := C.VkCommandBufferAllocateInfo{
		sType:              C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
		commandPool:        commandPool,
		level:              C.VK_COMMAND_BUFFER_LEVEL_PRIMARY,
		commandBufferCount: C.uint(len(commandBuffers)),
	}
	if C.vkAllocateCommandBuffers(device, &allocInfo, &commandBuffers[0]) != C.VK_SUCCESS {
		panic("vkAllocateCommandBuffers failed")
	}

	for i, commandBuffer := range commandBuffers {
		beginInfo := C.VkCommandBufferBeginInfo{
			sType: C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
			// flags:            0,   // Optional
			// pInheritanceInfo: nil, // Optional
		}

		if C.vkBeginCommandBuffer(commandBuffer, &beginInfo) != C.VK_SUCCESS {
			panic("vkBeginCommandBuffer failed")
		}

		renderPassInfo := C.VkRenderPassBeginInfo{
			sType:       C.VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO,
			renderPass:  renderPass,
			framebuffer: framebuffers[i],
			renderArea: C.VkRect2D{
				offset: C.VkOffset2D{0, 0},
				extent: swapchain.Extent,
			},
		}

		// clearColor := C.VkClearValue{0.0, 0.0, 0.0, 1.0}
		renderPassInfo.clearValueCount = 1
		renderPassInfo.pClearValues = C.defaultClearColor

		C.vkCmdBeginRenderPass(commandBuffer, &renderPassInfo, C.VK_SUBPASS_CONTENTS_INLINE)

		C.vkCmdBindPipeline(commandBuffer, C.VK_PIPELINE_BIND_POINT_GRAPHICS, graphicsPipeline)

		C.vkCmdDraw(commandBuffer, 3, 1, 0, 0)

		C.vkCmdEndRenderPass(commandBuffer)

		if C.vkEndCommandBuffer(commandBuffer) != C.VK_SUCCESS {
			panic("vkEndCommandBuffer failed")
		}
	}

	return commandBuffers
}

func createCommandPool(device C.VkDevice, indices QueueFamilyIndices) C.VkCommandPool {
	poolInfo := C.VkCommandPoolCreateInfo{
		sType:            C.VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
		queueFamilyIndex: indices.graphicsFamily,
		flags:            0, // Optional
	}
	var commandPool C.VkCommandPool
	if C.vkCreateCommandPool(device, &poolInfo, nil, &commandPool) != C.VK_SUCCESS {
		panic("vkCreateCommandPool failed")
	}
	return commandPool
}

func createFramebuffers(device C.VkDevice, swapchain Swapchain, renderPass C.VkRenderPass) []C.VkFramebuffer {
	framebuffers := make([]C.VkFramebuffer, len(swapchain.ImageViews))
	for i, iv := range swapchain.ImageViews {
		n := 1
		pAttachments := (*C.VkImageView)(C.malloc(C.size_t(n) * C.sizeof_VkImageView))
		qAttachments := (*[1 << 30]C.VkImageView)(unsafe.Pointer(pAttachments))[:n:n]
		qAttachments[0] = iv

		framebufferInfo := (*C.VkFramebufferCreateInfo)(C.malloc(C.sizeof_VkFramebufferCreateInfo))
		*framebufferInfo = C.VkFramebufferCreateInfo{
			sType:           C.VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO,
			renderPass:      renderPass,
			attachmentCount: 1,
			pAttachments:    pAttachments,
			width:           swapchain.Extent.width,
			height:          swapchain.Extent.height,
			layers:          1,
		}

		if C.vkCreateFramebuffer(device, framebufferInfo, nil, &framebuffers[i]) != C.VK_SUCCESS {
			panic("vkCreateFramebuffer failed")
		}
	}

	fmt.Println("framebuffers:", len(framebuffers))
	return framebuffers
}

func createRenderPass(device C.VkDevice, swapchain Swapchain) C.VkRenderPass {
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
	if C.vkCreateRenderPass(device, &renderPassInfo, nil, &renderPass) != C.VK_SUCCESS {
		panic("vkCreateRenderPass failed")
	}
	return renderPass
}

type ShaderModule struct {
	vkShaderModule C.VkShaderModule
}

func (sm ShaderModule) Destroy(dev Device) {
	C.vkDestroyShaderModule(dev.vkDevice, sm.vkShaderModule, nil)
}

type ShaderAsset string

func (name ShaderAsset) Module(dev Device) ShaderModule {
	return ShaderModule{createShaderModule(dev.vkDevice, mustReadFile(string(name)))}
}

func createShaderModule(device C.VkDevice, code []byte) C.VkShaderModule {
	// TODO this may not be appropriate, passing in bytes allocated in go to c,
	// but i think it should be ok, just double check at some point
	createInfo := (*C.VkShaderModuleCreateInfo)(C.malloc(C.sizeof_VkShaderModuleCreateInfo))
	*createInfo = C.VkShaderModuleCreateInfo{
		sType:    C.VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
		codeSize: C.size_t(len(code)),
		pCode:    (*C.uint)(unsafe.Pointer(&code[0])),
	}

	var shaderModule C.VkShaderModule
	if C.vkCreateShaderModule(device, createInfo, nil, &shaderModule) != C.VK_SUCCESS {
		panic("vkCreateShaderModule failed")
	}
	return shaderModule
}

func createGraphicsPipeline(device Device, swapchain Swapchain, renderPass C.VkRenderPass) (C.VkPipeline, C.VkPipelineLayout) {
	vertAsset := ShaderAsset("shaders/vert.spv")
	fragAsset := ShaderAsset("shaders/frag.spv")

	// vertShaderCode := mustReadFile("shaders/vert.spv")
	// fragShaderCode := mustReadFile("shaders/frag.spv")

	// vertShaderModule := createShaderModule(device, vertShaderCode)
	// defer C.vkDestroyShaderModule(device, vertShaderModule, nil)
	// fragShaderModule := createShaderModule(device, fragShaderCode)
	// defer C.vkDestroyShaderModule(device, fragShaderModule, nil)

	vertModule := vertAsset.Module(device)
	fragModule := fragAsset.Module(device)

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

	// maybe set this up elsewhere with matching defer destroy
	var pipelineLayout C.VkPipelineLayout
	pipelineLayoutInfo := C.VkPipelineLayoutCreateInfo{
		sType:                  C.VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
		setLayoutCount:         0,   // Optional
		pSetLayouts:            nil, // Optional
		pushConstantRangeCount: 0,   // Optional
		pPushConstantRanges:    nil, // Optional
	}
	if C.vkCreatePipelineLayout(device.vkDevice, &pipelineLayoutInfo, nil, &pipelineLayout) != C.VK_SUCCESS {
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

		layout: pipelineLayout,

		renderPass: renderPass,
		subpass:    0,

		// basePipelineHandle: VK_NULL_HANDLE, // Optional
		// basePipelineIndex:  -1,             // Optional
	}

	var graphicsPipeline C.VkPipeline
	if C.vkCreateGraphicsPipelines(device.vkDevice, C.VkPipelineCache(vkNullHandle), 1, &pipelineInfo, nil, &graphicsPipeline) != C.VK_SUCCESS {
		panic("vkCreateGraphicsPipelines failed")
	}

	return graphicsPipeline, pipelineLayout
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

func (device Device) CreateSwapchain() (Swapchain, error) {
	swapchain := Swapchain{
		device: device,
	}
	if err := swapchain.Create(device); err != nil {
		return Swapchain{}, err
	}
	return swapchain, nil
}

func (device Device) CreateSemaphore() Semaphore {
	s := Semaphore{vkDevice: device.vkDevice}
	if err := s.Create(); err != nil {
		panic(err)
	}
	return s
}

// *******************
// Swapchain
// *******************

type SwapchainSupport struct {
	Capabilities C.VkSurfaceCapabilitiesKHR
	Formats      []C.VkSurfaceFormatKHR
	PresentModes []C.VkPresentModeKHR
}

func GetSwapchainSupport(device C.VkPhysicalDevice, surface C.VkSurfaceKHR) SwapchainSupport {
	var ss SwapchainSupport
	C.vkGetPhysicalDeviceSurfaceCapabilitiesKHR(device, surface, &ss.Capabilities)

	var nformats C.uint
	C.vkGetPhysicalDeviceSurfaceFormatsKHR(device, surface, &nformats, nil)
	if nformats > 0 {
		p := (*C.VkSurfaceFormatKHR)(C.malloc(C.size_t(nformats) * C.sizeof_VkSurfaceFormatKHR))
		C.vkGetPhysicalDeviceSurfaceFormatsKHR(device, surface, &nformats, p)
		ss.Formats = (*[1 << 30]C.VkSurfaceFormatKHR)(unsafe.Pointer(p))[:nformats:nformats]
	}

	var nmodes C.uint
	C.vkGetPhysicalDeviceSurfacePresentModesKHR(device, surface, &nmodes, nil)
	if nmodes > 0 {
		p := (*C.VkPresentModeKHR)(C.malloc(C.size_t(nmodes) * C.sizeof_VkPresentModeKHR))
		C.vkGetPhysicalDeviceSurfacePresentModesKHR(device, surface, &nmodes, p)
		ss.PresentModes = (*[1 << 30]C.VkPresentModeKHR)(unsafe.Pointer(p))[:nmodes:nmodes]
	}

	return ss
}

func (details SwapchainSupport) ChooseSurfaceFormat() C.VkSurfaceFormatKHR {
	for _, f := range details.Formats {
		if f.format == C.VK_FORMAT_B8G8R8A8_SRGB && f.colorSpace == C.VK_COLOR_SPACE_SRGB_NONLINEAR_KHR {
			return f
		}
	}
	return details.Formats[0]
}

func (details SwapchainSupport) ChoosePresentMode() C.VkPresentModeKHR {
	for _, m := range details.PresentModes {
		if m == C.VK_PRESENT_MODE_MAILBOX_KHR {
			return m
		}
	}
	return C.VK_PRESENT_MODE_FIFO_KHR
	// sometimes the above is buggy and below is preferred, haven't gotten far enough to know myself
	// apparently being buggy is outdated, still keep in mind the below
	// return VK_PRESENT_MODE_IMMEDIATE_KHR
}

func (ss SwapchainSupport) ChooseExtent() C.VkExtent2D {
	// The swap extent is the resolution of the swap chain images
	if ss.Capabilities.currentExtent.width != math.MaxUint32 {
		return ss.Capabilities.currentExtent
	}

	extent := C.VkExtent2D{Width, Height}
	extent.width = cclampu32(extent.width, ss.Capabilities.minImageExtent.width, ss.Capabilities.maxImageExtent.width)
	extent.height = cclampu32(extent.height, ss.Capabilities.minImageExtent.height, ss.Capabilities.maxImageExtent.height)
	return extent
}

type Swapchain struct {
	device      Device
	vkSwapchain C.VkSwapchainKHR

	ImageFormat C.VkFormat
	Extent      C.VkExtent2D

	images     []C.VkImage
	ImageViews []C.VkImageView
}

func (sc *Swapchain) Create(device Device) error {
	ss := GetSwapchainSupport(device.vkPhysicalDevice, device.surface.vkSurface)

	surfaceFormat := ss.ChooseSurfaceFormat()
	presentMode := ss.ChoosePresentMode()
	extent := ss.ChooseExtent()

	// take the minimum plus one so we do not ahve to wait on driver to complete internal ops
	// before we can acquire and render to another image
	imageCount := ss.Capabilities.minImageCount + 1
	if ss.Capabilities.maxImageCount > 0 && imageCount > ss.Capabilities.maxImageCount {
		imageCount = ss.Capabilities.maxImageCount
	}

	fmt.Println("capabilities.minImageCount:", ss.Capabilities.minImageCount)
	fmt.Println("capabilities.maxImageCount:", ss.Capabilities.maxImageCount)
	fmt.Println("swapchainCreateInfo.imageCount:", imageCount)

	createInfo := C.VkSwapchainCreateInfoKHR{
		sType:            C.VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR,
		surface:          device.surface.vkSurface,
		minImageCount:    imageCount,
		imageFormat:      surfaceFormat.format,
		imageColorSpace:  surfaceFormat.colorSpace,
		imageExtent:      extent,
		imageArrayLayers: 1, // always 1 unless making stereoscopic 3d app
		imageUsage:       C.VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
	}

	queueFamilyIndices := []C.uint{device.indices.graphicsFamily, device.indices.presentFamily}

	// TODO first case breaks b/c im not properly creating separate queues yet given i skipped over it
	if device.indices.graphicsFamily != device.indices.presentFamily {
		createInfo.imageSharingMode = C.VK_SHARING_MODE_CONCURRENT
		createInfo.queueFamilyIndexCount = 2
		createInfo.pQueueFamilyIndices = &queueFamilyIndices[0]
	} else {
		createInfo.imageSharingMode = C.VK_SHARING_MODE_EXCLUSIVE
		createInfo.queueFamilyIndexCount = 0
		createInfo.pQueueFamilyIndices = nil
	}

	createInfo.preTransform = ss.Capabilities.currentTransform

	createInfo.compositeAlpha = C.VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR

	createInfo.presentMode = presentMode
	createInfo.clipped = C.VK_TRUE

	// TODO don't know how i'd possibly set this ..
	// createInfo.oldSwapchain = C.VK_NULL_HANDLE

	sc.ImageFormat = surfaceFormat.format
	sc.Extent = extent

	if C.vkCreateSwapchainKHR(sc.device.vkDevice, &createInfo, nil, &sc.vkSwapchain) != C.VK_SUCCESS {
		return fmt.Errorf("vkCreateSwapchainKHR failed")
	}
	return nil
}

func (sc *Swapchain) getImages() {
	var n C.uint
	C.vkGetSwapchainImagesKHR(sc.device.vkDevice, sc.vkSwapchain, &n, nil)
	p := (*C.VkImage)(C.malloc(C.size_t(n) * C.sizeof_VkImage))
	C.vkGetSwapchainImagesKHR(sc.device.vkDevice, sc.vkSwapchain, &n, p)
	sc.images = (*[1 << 30]C.VkImage)(unsafe.Pointer(p))[:n:n]
	fmt.Println("len(sc.images):", len(sc.images))
}

func (sc *Swapchain) CreateImageViews() {
	if len(sc.images) == 0 {
		sc.getImages()
	}

	sc.ImageViews = make([]C.VkImageView, len(sc.images))
	for i := range sc.ImageViews {
		createInfo := C.VkImageViewCreateInfo{
			sType:    C.VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
			image:    sc.images[i],
			viewType: C.VK_IMAGE_VIEW_TYPE_2D,
			format:   sc.ImageFormat,
			components: C.VkComponentMapping{
				r: C.VK_COMPONENT_SWIZZLE_IDENTITY,
				g: C.VK_COMPONENT_SWIZZLE_IDENTITY,
				b: C.VK_COMPONENT_SWIZZLE_IDENTITY,
				a: C.VK_COMPONENT_SWIZZLE_IDENTITY,
			},
			subresourceRange: C.VkImageSubresourceRange{
				aspectMask:     C.VK_IMAGE_ASPECT_COLOR_BIT,
				baseMipLevel:   0,
				levelCount:     1,
				baseArrayLayer: 0,
				layerCount:     1,
			},
		}

		if C.vkCreateImageView(sc.device.vkDevice, &createInfo, nil, &sc.ImageViews[i]) != C.VK_SUCCESS {
			panic("vkCreateImageView failed")
		}
	}
	fmt.Println("created imageviews")
}

func (sc Swapchain) Destroy() {
	C.vkDestroySwapchainKHR(sc.device.vkDevice, sc.vkSwapchain, nil)
}

func (sc Swapchain) DestroyImageViews() {
	for _, iv := range sc.ImageViews {
		C.vkDestroyImageView(sc.device.vkDevice, iv, nil)
	}
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
	bin, err := ioutil.ReadFile(filename)
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
