(function (global) {
    "use strict";

    var timeTagRegex = /\[(\d{1,2}):(\d{2})(?:\.(\d{1,3}))?\]/g;
    var metadataTagRegex = /^\[[a-z]{1,8}:[^\]]*\]$/i;
    var sectionTagRegex = /^\[((?:verse|chorus|bridge|intro|outro|pre-chorus|post-chorus|hook|refrain|interlude))\]$/i;

    function fractionToMilliseconds(fractionText) {
        if (!fractionText) return 0;
        var normalized = String(fractionText);
        while (normalized.length < 3) normalized += "0";
        return parseInt(normalized, 10) || 0;
    }

    function parseTimeTag(match) {
        var minutes = parseInt(match[1], 10);
        var seconds = parseInt(match[2], 10);
        if (isNaN(minutes) || isNaN(seconds) || seconds < 0 || seconds >= 60) {
            return null;
        }
        return minutes * 60 * 1000 + seconds * 1000 + fractionToMilliseconds(match[3]);
    }

    function parseLRC(text) {
        var lines = String(text || "").split(/\r?\n/);
        var parsed = [];

        lines.forEach(function (rawLine) {
            var line = rawLine || "";
            var matches = [];
            var match;
            while ((match = timeTagRegex.exec(line)) !== null) {
                matches.push(match);
            }
            timeTagRegex.lastIndex = 0;

            var cleanText = line.replace(timeTagRegex, "").trim();
            if (!matches.length) {
                if (!cleanText || metadataTagRegex.test(cleanText)) return;
                var sectionMatch = cleanText.match(sectionTagRegex);
                parsed.push({
                    timeMs: null,
                    text: sectionMatch ? sectionMatch[1] : cleanText,
                    isSectionLabel: !!sectionMatch
                });
                return;
            }
            if (!cleanText) return;

            matches.forEach(function (timeMatch) {
                var timeMs = parseTimeTag(timeMatch);
                if (timeMs == null) return;
                parsed.push({timeMs: timeMs, text: cleanText, isSectionLabel: false});
            });
        });

        return parsed;
    }

    function isSyncedLRC(text) {
        var lines = parseLRC(text);
        for (var i = 0; i < lines.length; i++) {
            if (typeof lines[i].timeMs === "number") return true;
        }
        return false;
    }

    function findActiveIndex(lines, currentTimeMs, lastIndex) {
        if (!Array.isArray(lines) || !lines.length) return -1;
        var current = Math.max(0, Math.floor(Number(currentTimeMs) || 0));
        var index = typeof lastIndex === "number" ? lastIndex : -1;
        if (index >= lines.length) index = lines.length - 1;

        if (index >= 0 && lines[index] && typeof lines[index].timeMs === "number") {
            if (lines[index].timeMs > current) {
                while (index >= 0) {
                    if (typeof lines[index].timeMs === "number" && lines[index].timeMs <= current) {
                        return index;
                    }
                    index--;
                }
                return -1;
            }
            while (index + 1 < lines.length) {
                var next = lines[index + 1];
                if (typeof next.timeMs !== "number") {
                    index++;
                    continue;
                }
                if (next.timeMs <= current) {
                    index++;
                    continue;
                }
                break;
            }
            return typeof lines[index].timeMs === "number" ? index : -1;
        }

        var active = -1;
        for (var i = 0; i < lines.length; i++) {
            if (typeof lines[i].timeMs === "number" && lines[i].timeMs <= current) {
                active = i;
            } else if (typeof lines[i].timeMs === "number") {
                break;
            }
        }
        return active;
    }

    global.SonicLRC = {
        parseLRC: parseLRC,
        isSyncedLRC: isSyncedLRC,
        findActiveIndex: findActiveIndex
    };
})(window);
