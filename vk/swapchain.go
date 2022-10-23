package vk

/*
#define GLFW_INCLUDE_VULKAN
#include <GLFW/glfw3.h>
#include <stdlib.h>
#include "vk.h"
*/
import "C"

import (
	"fmt"
	"math"
	"unsafe"
)

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

func (ss SwapchainSupport) ChooseExtent(width, height uint) C.VkExtent2D {
	// The swap extent is the resolution of the swap chain images
	if ss.Capabilities.currentExtent.width != math.MaxUint32 {
		return ss.Capabilities.currentExtent
	}

	extent := C.VkExtent2D{C.uint(width), C.uint(height)}
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

	Framebuffers []C.VkFramebuffer

	renderPass C.VkRenderPass
	graphicsPipeline Pipeline

	commandBuffers []C.VkCommandBuffer
}

func (sc *Swapchain) Create(width, height uint) error {
	ss := GetSwapchainSupport(sc.device.vkPhysicalDevice, sc.device.surface.vkSurface)

	surfaceFormat := ss.ChooseSurfaceFormat()
	presentMode := ss.ChoosePresentMode()
	extent := ss.ChooseExtent(width, height)

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
		surface:          sc.device.surface.vkSurface,
		minImageCount:    imageCount,
		imageFormat:      surfaceFormat.format,
		imageColorSpace:  surfaceFormat.colorSpace,
		imageExtent:      extent,
		imageArrayLayers: 1, // always 1 unless making stereoscopic 3d app
		imageUsage:       C.VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
	}

	queueFamilyIndices := []C.uint{sc.device.indices.graphicsFamily, sc.device.indices.presentFamily}

	// TODO first case breaks b/c im not properly creating separate queues yet given i skipped over it
	if sc.device.indices.graphicsFamily != sc.device.indices.presentFamily {
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

	// TODO Go 1.17 allows conversions from slice to array pointer. Consider if that makes sense here keeping in mind where alloc occurs
	p := (*C.VkImage)(C.malloc(C.size_t(n) * C.sizeof_VkImage))

	C.vkGetSwapchainImagesKHR(sc.device.vkDevice, sc.vkSwapchain, &n, p)

	// TODO Go 1.17 adds unsafe.Slice, which should allow the following
	// sc.images = unsafe.Slice(p, n)
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

func (sc *Swapchain) CreateFramebuffers() {
	sc.Framebuffers = make([]C.VkFramebuffer, len(sc.ImageViews))
	for i, iv := range sc.ImageViews {
		n := 1
		pAttachments := (*C.VkImageView)(C.malloc(C.size_t(n) * C.sizeof_VkImageView))
		qAttachments := (*[1 << 30]C.VkImageView)(unsafe.Pointer(pAttachments))[:n:n]
		qAttachments[0] = iv

		framebufferInfo := (*C.VkFramebufferCreateInfo)(C.malloc(C.sizeof_VkFramebufferCreateInfo))
		*framebufferInfo = C.VkFramebufferCreateInfo{
			sType:           C.VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO,
			renderPass:      sc.renderPass,
			attachmentCount: 1,
			pAttachments:    pAttachments,
			width:           sc.Extent.width,
			height:          sc.Extent.height,
			layers:          1,
		}

		if C.vkCreateFramebuffer(sc.device.vkDevice, framebufferInfo, nil, &sc.Framebuffers[i]) != C.VK_SUCCESS {
			panic("vkCreateFramebuffer failed")
		}
	}

	fmt.Println("framebuffers:", len(sc.Framebuffers))
}

func (sc *Swapchain) CreateRenderPass() {
	sc.renderPass = sc.device.CreateRenderPass(sc)
}

func (sc *Swapchain) CreateGraphicsPipeline() {
	sc.graphicsPipeline = sc.device.CreateGraphicsPipeline(sc, sc.renderPass)
}

func (sc *Swapchain) CreateCommandBuffers() {
	sc.commandBuffers = make([]C.VkCommandBuffer, len(sc.Framebuffers))

	/*
	The level parameter specifies if the allocated command buffers are primary or secondary command buffers.

	    VK_COMMAND_BUFFER_LEVEL_PRIMARY: Can be submitted to a queue for execution, but cannot be called from other command buffers.
	    VK_COMMAND_BUFFER_LEVEL_SECONDARY: Cannot be submitted directly, but can be called from primary command buffers.

	We won't make use of the secondary command buffer functionality here, but you can imagine that it's helpful to reuse common operations from primary command buffers.
	*/
	allocInfo := C.VkCommandBufferAllocateInfo{
		sType:              C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
		commandPool:        sc.device.commandPool,
		level:              C.VK_COMMAND_BUFFER_LEVEL_PRIMARY,
		commandBufferCount: C.uint(len(sc.commandBuffers)),
	}
	if C.vkAllocateCommandBuffers(sc.device.vkDevice, &allocInfo, &sc.commandBuffers[0]) != C.VK_SUCCESS {
		panic("vkAllocateCommandBuffers failed")
	}
	// command buffers are freed automatically when parent command pool is freed

	// ******************************************************************
	// ************************* Draw Commands **************************
	// ******************************************************************

	for i, commandBuffer := range sc.commandBuffers {
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
			renderPass:  sc.renderPass,
			framebuffer: sc.Framebuffers[i],
			renderArea: C.VkRect2D{
				offset: C.VkOffset2D{0, 0},
				extent: sc.Extent,
			},
		}

		// clearColor := C.VkClearValue{0.0, 0.0, 0.0, 1.0}
		renderPassInfo.clearValueCount = 1
		renderPassInfo.pClearValues = C.defaultClearColor

		C.vkCmdBeginRenderPass(commandBuffer, &renderPassInfo, C.VK_SUBPASS_CONTENTS_INLINE)

		C.vkCmdBindPipeline(commandBuffer, C.VK_PIPELINE_BIND_POINT_GRAPHICS, sc.graphicsPipeline.vkPipeline)

		C.vkCmdDraw(commandBuffer, 3, 1, 0, 0)

		C.vkCmdEndRenderPass(commandBuffer)

		if C.vkEndCommandBuffer(commandBuffer) != C.VK_SUCCESS {
			panic("vkEndCommandBuffer failed")
		}

		/*
		commandBuffer.Begin()
		renderPass.Begin()
		pipeline.Bind(POINT_GRAPHICS)
		commandBuffer.Draw(3, 1, 0, 0)
		renderpass.End()
		commandBuffer.End()
		*/

		/*
		cmd.BeginCommandBuffer()
		cmd.BeginRenderPass()
		cmd.BindPipeline()
		cmd.Draw(3, 1, 0, 0)
		cmd.EndRenderPass()
		cmd.EndCommandBuffer()
		*/

		/*
		cbuf := pool.Get()
		cbuf
		*/
	}
}

func (sc *Swapchain) Destroy() {
	for _, fbuf := range sc.Framebuffers {
		C.vkDestroyFramebuffer(sc.device.vkDevice, fbuf, nil)
	}

	// free command buffers
	C.vkFreeCommandBuffers(sc.device.vkDevice, sc.device.commandPool, C.uint(len(sc.commandBuffers)), &sc.commandBuffers[0])

	sc.graphicsPipeline.Destroy()
	C.vkDestroyRenderPass(sc.device.vkDevice, sc.renderPass, nil)

	for _, iv := range sc.ImageViews {
		C.vkDestroyImageView(sc.device.vkDevice, iv, nil)
	}

	C.vkDestroySwapchainKHR(sc.device.vkDevice, sc.vkSwapchain, nil)

	sc.images = nil
}