#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#import <objc/runtime.h>
#import "Enums.h"
#import "MRContent.h"
//#import <Cocoa/Cocoa.h>

// https://github.com/billziss-gh/EnergyBar/blob/master/src/System/NowPlaying.m#L21
typedef void (^MRMediaRemoteGetNowPlayingInfoBlock)(NSDictionary *info);
typedef void (^MRMediaRemoteGetNowPlayingClientBlock)(id clientObj);
typedef void (^MRMediaRemoteGetNowPlayingApplicationIsPlayingBlock)(BOOL playing);

typedef void (*MRMediaRemoteGetNowPlayingInfoFunction)(dispatch_queue_t queue, MRMediaRemoteGetNowPlayingInfoBlock);
typedef void (*MRMediaRemoteGetNowPlayingClientFunction)(dispatch_queue_t queue, MRMediaRemoteGetNowPlayingClientBlock);
typedef void (*MRMediaRemoteGetNowPlayingApplicationIsPlayingFunction) (dispatch_queue_t queue,MRMediaRemoteGetNowPlayingApplicationIsPlayingBlock);
typedef void (*MRMediaRemoteSetElapsedTimeFunction)(double time);
typedef Boolean (*MRMediaRemoteSendCommandFunction)(MRMediaRemoteCommand cmd, NSDictionary* userInfo);
typedef NSString* (*MRNowPlayingClientGetBundleIdentifierFunction)(id);


void printHelp(void) {
    printf("Example Usage: \n");
    printf("\tnowplaying-cli get-raw\n");
    printf("\tnowplaying-cli get title album artist\n");
    printf("\tnowplaying-cli pause\n");
    printf("\tnowplaying-cli seek 60\n");
    printf("\n");
    printf("Available commands: \n");
    printf("\tget, get-raw, play, pause, togglePlayPause, next, previous, seek <secs>\n");
}

typedef enum {
    GET,
    GET_RAW,
    MEDIA_COMMAND,
    SEEK,

} Command;

NSDictionary<NSString*, NSNumber*> *cmdTranslate = @{
    @"play": @(MRMediaRemoteCommandPlay),
    @"pause": @(MRMediaRemoteCommandPause),
    @"togglePlayPause": @(MRMediaRemoteCommandTogglePlayPause),
    @"next": @(MRMediaRemoteCommandNextTrack),
    @"previous": @(MRMediaRemoteCommandPreviousTrack),
};

int main(int argc, char** argv) {

    if(argc == 1) {
        printHelp();
        return 0;
    }

    Command command = GET;
    NSString *cmdStr = [NSString stringWithUTF8String:argv[1]];
    double seekTime = 0;

    int numKeys = argc - 2;
    NSMutableArray<NSString *> *keys = [NSMutableArray array];
    if(strcmp(argv[1], "get") == 0) {
        for(int i = 2; i < argc; i++) {
            NSString *key = [NSString stringWithUTF8String:argv[i]];
            [keys addObject:key];
        }
        command = GET;
    }
    else if(strcmp(argv[1], "get-raw") == 0) {
        command = GET_RAW;
    }
    else if(strcmp(argv[1], "seek") == 0 && argc == 3) {
        command = SEEK;
        char *end;
        seekTime = strtod(argv[2], &end);
        if(*end != '\0') {
            fprintf(stderr, "Invalid seek time: %s\n", argv[2]);
            fprintf(stderr, "Usage: nowplaying-cli seek <secs>\n");
            return 1;
        }
    }
    else if(cmdTranslate[cmdStr] != nil) {
        command = MEDIA_COMMAND;
    }
    else {
        printHelp();
        return 0;
    }

    NSAutoreleasePool* pool = [[NSAutoreleasePool alloc] init];
    NSPanel* panel = [[NSPanel alloc]
        initWithContentRect: NSMakeRect(0, 0, 0, 0)
        styleMask: NSWindowStyleMaskTitled
        backing: NSBackingStoreBuffered
        defer: NO];

    
    CFURLRef ref = (__bridge CFURLRef) [NSURL fileURLWithPath:@"/System/Library/PrivateFrameworks/MediaRemote.framework"];
    CFBundleRef bundle = CFBundleCreate(kCFAllocatorDefault, ref);

    MRMediaRemoteSendCommandFunction MRMediaRemoteSendCommand = (MRMediaRemoteSendCommandFunction) CFBundleGetFunctionPointerForName(bundle, CFSTR("MRMediaRemoteSendCommand"));
    if(command == MEDIA_COMMAND) {
        MRMediaRemoteSendCommand((MRMediaRemoteCommand) [cmdTranslate[cmdStr] intValue], nil);
    }

    MRMediaRemoteSetElapsedTimeFunction MRMediaRemoteSetElapsedTime = (MRMediaRemoteSetElapsedTimeFunction) CFBundleGetFunctionPointerForName(bundle, CFSTR("MRMediaRemoteSetElapsedTime"));
    if(command == SEEK) {
        MRMediaRemoteSetElapsedTime(seekTime);
    }
   
    MRMediaRemoteGetNowPlayingApplicationIsPlayingFunction MRMediaRemoteGetNowPlayingApplicationIsPlaying = (MRMediaRemoteGetNowPlayingApplicationIsPlayingFunction)CFBundleGetFunctionPointerForName(bundle,CFSTR("MRMediaRemoteGetNowPlayingApplicationIsPlaying"));
    
     //判断是否正在播放正在播放的
    //BOOL isPlaying =NO;
    MRMediaRemoteGetNowPlayingApplicationIsPlaying(dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_HIGH, 0), ^(BOOL playing) {
        printf("%s\n", playing ? "YES" : "NO");
        //isPlaying=playing;
        if(playing == NO) {
            [NSApp terminate:nil];
           return;
       }
    [NSApp terminate:nil];
    });
  
    
    // 获取 MRMediaRemoteGetNowPlayingClient 实例
    MRMediaRemoteGetNowPlayingClientFunction MRMediaRemoteGetNowPlayingClient = (MRMediaRemoteGetNowPlayingClientFunction)CFBundleGetFunctionPointerForName(bundle,CFSTR("MRMediaRemoteGetNowPlayingClient"));
    
    MRMediaRemoteGetNowPlayingClient(dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_DEFAULT, 0), ^(id clientObj) {
        // 获取roon
        // MRNowPlayingClientGetBundleIdentifier
        MRNowPlayingClientGetBundleIdentifierFunction MRNowPlayingClientGetBundleIdentifier = (MRNowPlayingClientGetBundleIdentifierFunction)CFBundleGetFunctionPointerForName(bundle,CFSTR("MRNowPlayingClientGetBundleIdentifier"));
        NSString *bundleIdentifier = MRNowPlayingClientGetBundleIdentifier(clientObj);
        printf("%s\n", [bundleIdentifier UTF8String]);
        [NSApp terminate:nil];
    });

   

    MRMediaRemoteGetNowPlayingInfoFunction MRMediaRemoteGetNowPlayingInfo = (MRMediaRemoteGetNowPlayingInfoFunction) CFBundleGetFunctionPointerForName(bundle, CFSTR("MRMediaRemoteGetNowPlayingInfo"));
    MRMediaRemoteGetNowPlayingInfo(dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_LOW, 0), ^(NSDictionary* information) {
        if(command == MEDIA_COMMAND || command == SEEK) {
            [NSApp terminate:nil];
            return;
        }

        NSString *data = [information description];
        const char *dataStr = [data UTF8String];
        if(command == GET_RAW) {
            printf("%s\n", dataStr);
            [NSApp terminate:nil];
            return;
        }

        for(int i = 0; i < numKeys; i++) {
            NSString *propKey = [keys[i] stringByReplacingCharactersInRange:NSMakeRange(0,1) withString:[[keys[i] substringToIndex:1] capitalizedString]];
            NSString *key = [NSString stringWithFormat:@"kMRMediaRemoteNowPlayingInfo%@", propKey];
            NSObject *rawValue = [information objectForKey:key];
            if(rawValue == nil) {
                printf("null\n");
            }
            else if([key isEqualToString:@"kMRMediaRemoteNowPlayingInfoArtworkData"] || [key isEqualToString:@"kMRMediaRemoteNowPlayingInfoClientPropertiesData"]) {
                NSData *data = (NSData *) rawValue;
                NSString *base64 = [data base64EncodedStringWithOptions:0];
                printf("%s\n", [base64 UTF8String]);
            }
            else if([key isEqualToString:@"kMRMediaRemoteNowPlayingInfoElapsedTime"]) {
                MRContentItem *item = [[objc_getClass("MRContentItem") alloc] initWithNowPlayingInfo:(__bridge NSDictionary *)information];
                NSString *value = [NSString stringWithFormat:@"%f", item.metadata.calculatedPlaybackPosition];
                const char *valueStr = [value UTF8String];
                printf("%s\n", valueStr);
            }
            else {
                NSString *value = [NSString stringWithFormat:@"%@", rawValue];
                const char *valueStr = [value UTF8String];
                printf("%s\n", valueStr);
            }
        }
        [NSApp terminate:nil];
    });

    [NSApp run];
    [pool release];
    return 0;
}
//     symbols:         [ _MRMediaRemoteRegisterForNowPlayingNotifications, _MRMediaRemoteUnregisterForNowPlayingNotifications, _MRMediaRemoteGetNowPlayingClient, _MRMediaRemoteGetNowPlayingInfo, _MRMediaRemoteGetNowPlayingApplicationIsPlaying, _MRNowPlayingClientGetBundleIdentifier, _MRNowPlayingClientGetParentAppBundleIdentifier, _MRMediaRemoteSetElapsedTime, _MRMediaRemoteSendCommand, _kMRMediaRemoteNowPlayingInfoDidChangeNotification, _kMRMediaRemoteNowPlayingPlaybackQueueDidChangeNotification, _kMRMediaRemotePickableRoutesDidChangeNotification, _kMRMediaRemoteNowPlayingApplicationDidChangeNotification, _kMRMediaRemoteNowPlayingApplicationIsPlayingDidChangeNotification, _kMRMediaRemoteRouteStatusDidChangeNotification, _kMRNowPlayingPlaybackQueueChangedNotification, _kMRPlaybackQueueContentItemsChangedNotification, _kMRMediaRemoteNowPlayingInfoArtist, _kMRMediaRemoteNowPlayingInfoTitle, _kMRMediaRemoteNowPlayingInfoAlbum, _kMRMediaRemoteNowPlayingInfoArtworkData, _kMRMediaRemoteNowPlayingInfoPlaybackRate, _kMRMediaRemoteNowPlayingInfoDuration, _kMRMediaRemoteNowPlayingInfoElapsedTime, _kMRMediaRemoteNowPlayingInfoTimestamp, _kMRMediaRemoteNowPlayingInfoClientPropertiesData, _kMRMediaRemoteNowPlayingInfoArtworkIdentifier, _kMRMediaRemoteNowPlayingInfoShuffleMode, _kMRMediaRemoteNowPlayingInfoTrackNumber, _kMRMediaRemoteNowPlayingInfoTotalQueueCount, _kMRMediaRemoteNowPlayingInfoArtistiTunesStoreAdamIdentifier, _kMRMediaRemoteNowPlayingInfoArtworkMIMEType, _kMRMediaRemoteNowPlayingInfoMediaType, _kMRMediaRemoteNowPlayingInfoiTunesStoreSubscriptionAdamIdentifier, _kMRMediaRemoteNowPlayingInfoGenre, _kMRMediaRemoteNowPlayingInfoComposer, _kMRMediaRemoteNowPlayingInfoQueueIndex, _kMRMediaRemoteNowPlayingInfoiTunesStoreIdentifier, _kMRMediaRemoteNowPlayingInfoTotalTrackCount, _kMRMediaRemoteNowPlayingInfoContentItemIdentifier, _kMRMediaRemoteNowPlayingInfoIsMusicApp, _kMRMediaRemoteNowPlayingInfoAlbumiTunesStoreAdamIdentifier, _kMRMediaRemoteNowPlayingInfoUniqueIdentifier, _kMRActiveNowPlayingPlayerPathUserInfoKey, _kMRMediaRemoteNowPlayingApplicationIsPlayingUserInfoKey, _kMRMediaRemoteNowPlayingApplicationDisplayNameUserInfoKey, _kMRMediaRemoteNowPlayingApplicationPIDUserInfoKey, _kMRMediaRemoteOriginUserInfoKey, _kMRMediaRemotePlaybackStateUserInfoKey, _kMRMediaRemoteUpdatedContentItemsUserInfoKey, _kMRNowPlayingClientUserInfoKey, _kMRNowPlayingPlayerPathUserInfoKey, _kMRNowPlayingPlayerUserInfoKey, _kMROriginActiveNowPlayingPlayerPathUserInfoKey ]

// kMRMediaRemoteNowPlayingInfoClientPropertiesData
// MRNowPlayingClientGetBundleIdentifier
