video.mp4:
	ffmpeg -r 60 -f image2 -s 2560x1440 -i output/%04d_julia.png -vcodec libx264 -crf 25  -pix_fmt yuv420p video.mp4

clean:
	rm -f video.mp4 output/*