CC = clang
CFLAGS = -O3
FRAMEWORKS = -framework Cocoa
INCLUDES = -I./include

nowplaying-cli-mac: main.m
	$(CC) $(CFLAGS) $(FRAMEWORKS) $(INCLUDES) $< -o $@

clean:
	rm -f nowplaying-cli-mac
