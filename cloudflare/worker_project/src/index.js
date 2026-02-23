// 简单的路由助手
class Router {
    constructor() {
        this.routes = [];
    }

    handle(request, env, ctx) {
        // 允许的域名列表
        const allowedHosts = [
            "blog-vincent.chyu.org",
            "vincent.chyu.org",     // 如果你有多个博客域名，就继续加.....
        ];

        // 获取 referer
        const referer = request.headers.get("Referer") || "";

        // referer 不存在（例如直接调用 API）——你要不要允许？通常是不允许
        if (!referer) {
            return new Response("Forbidden: missing referer", {status: 403});
        }

        // 判断 referer 是否包含允许的域名
        const isAllowed = allowedHosts.some(host => referer.includes(host));

        if (!isAllowed) {
            return new Response("Forbidden: invalid referer", {status: 403});
        }

        const url = new URL(request.url);

        const method = request.method;

        for (const route of this.routes) {
            if (route.method !== method && route.method !== 'ALL') continue;

            const match = route.pattern.exec(url.pathname);
            if (match) {
                request.params = match.groups || {};
                return route.handler(request, env, ctx);
            }
        }

        return new Response('Not Found', {status: 404});
    }

    get(path, handler) {
        const pattern = new RegExp('^' + path.replace(/:\w+/g, '(?<$&>\\w+)').replace(':', '') + '$');
        this.routes.push({method: 'GET', pattern, handler});
    }

    post(path, handler) {
        const pattern = new RegExp('^' + path.replace(/:\w+/g, '(?<$&>\\w+)').replace(':', '') + '$');
        this.routes.push({method: 'POST', pattern, handler});
    }

    options(path, handler) {
        // 处理通配符 .*
        let regexPath = path.replace(/:\w+/g, '(?<$&>\\w+)').replace(':', '');
        if (path.includes('.*')) {
            regexPath = path.replace('.*', '.*');
        } else {
            regexPath = '^' + regexPath + '$';
        }
        const pattern = new RegExp(regexPath);
        this.routes.push({method: 'OPTIONS', pattern, handler});
    }

    all(path, handler) {
        const pattern = new RegExp('^' + path.replace(/:\w+/g, '(?<$&>\\w+)').replace(':', '') + '$');
        this.routes.push({method: 'ALL', pattern, handler});
    }
}

const router = new Router();

// CORS Headers
const corsHeaders = {
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
    "Access-Control-Allow-Headers": "Content-Type",
    "Access-Control-Max-Age": "86400",
};

// 辅助响应函数
function jsonResponse(data, headers = {}) {
    return Response.json(data, {
        headers: {...corsHeaders, ...headers}
    });
}

function errorResponse(message, status = 500) {
    return Response.json({error: message}, {
        status,
        headers: corsHeaders
    });
}

// OPTIONS 处理
router.options('/api/.*', () => new Response(null, {headers: corsHeaders}));

// --- API Endpoints ---

// 缓存处理函数
async function withCache(request, env, handler, ttl = 300) {
    const cacheUrl = new URL(request.url);
    const cacheKey = new Request(cacheUrl.toString(), request);
    const cache = caches.default;

    let response = await cache.match(cacheKey);
    if (response) {
        console.log(`Cache HIT for ${request.url}`);
        return response;
    }

    console.log(`Cache MISS for ${request.url}`);
    response = await handler(request, env);

    // 只有 200 OK 才缓存
    if (response.status === 200) {
        // 克隆响应以便修改 headers
        response = new Response(response.body, response);
        response.headers.set("Cache-Control", `public, max-age=${ttl}`);
        // 写入缓存
        // waitUntil 并不是必需的，但在 worker 中为了不阻塞返回可以使用 ctx.waitUntil，
        // 不过 api 简单封装下直接 put 也可以。
        // 注意：Cache API 不支持 PUT 带有 "Set-Cookie" 的 Response，这里 API 应该没这个问题。
        await cache.put(cacheKey, response.clone());
    }
    return response;
}

// Dashboard Stats
router.get('/api/dashboard/stats', (req, env, ctx) => withCache(req, env, async (req, env) => {
    try {
        const db = env.DB;
        console.log("Accessing /api/dashboard/stats");
        if (!db) console.error("DB binding not found!");
        const [totalPlays, totalTracks, totalArtists, totalAlbums] = await Promise.all([
            db.prepare("SELECT SUM(play_count) as count FROM tracks").first(),
            db.prepare("SELECT COUNT(*) as count FROM tracks").first(),
            db.prepare("SELECT COUNT(DISTINCT artist) as count FROM tracks").first(),
            db.prepare("SELECT COUNT(DISTINCT album) as count FROM tracks").first()
        ]);

        const result = {
            totalPlays: totalPlays?.count || 0,
            totalTracks: totalTracks?.count || 0,
            totalArtists: totalArtists?.count || 0,
            totalAlbums: totalAlbums?.count || 0
        };

        console.log("Stats query result:", JSON.stringify(result));

        return jsonResponse(result);
    } catch (err) {
        return errorResponse(err.message);
    }
}, 300));

// Play Counts by Source
router.get('/api/dashboard/play-counts-by-source', (req, env, ctx) => withCache(req, env, async (req, env) => {
    try {
        const db = env.DB;
        const {results} = await db.prepare("SELECT source, count(source) as count FROM track_play_records GROUP BY source").all();
        const response = {};
        results.forEach(row => {
            let key = row.source || "Unknown";
            if (key.toLowerCase() === "apple music" || key === "applemusic") key = "Apple Music";
            else if (key.toLowerCase() === "audirvana") key = "Audirvana";
            else if (key.toLowerCase() === "roon") key = "Roon";
            response[key] = (response[key] || 0) + row.count;
        });
        return jsonResponse(response);
    } catch (err) {
        return errorResponse(err.message);
    }
}, 300));

// Trend
router.get('/api/dashboard/trend', (req, env, ctx) => withCache(req, env, async (req, env) => {
    try {
        const db = env.DB;
        const url = new URL(req.url);
        const range = parseInt(url.searchParams.get("range") || "30");
        const startDate = new Date();
        startDate.setDate(startDate.getDate() - range);
        const startDateStr = startDate.toISOString();

        const {results} = await db.prepare(`
            SELECT strftime('%Y-%m-%d', datetime(play_time, '+8 hours')) as date, 
              strftime('%H', datetime(play_time, '+8 hours')) as hour, 
              COUNT(*) as count
            FROM track_play_records
            WHERE play_time >= ?
            GROUP BY date, hour
            ORDER BY date, hour
        `).bind(startDateStr).all();

        const hourlyData = {};
        results.forEach(row => {
            if (!hourlyData[row.date]) {
                hourlyData[row.date] = {total: 0, hourly: {}};
                for (let i = 0; i < 24; i++) hourlyData[row.date].hourly[i.toString().padStart(2, '0')] = 0;
            }
            hourlyData[row.date].hourly[row.hour] = row.count;
            hourlyData[row.date].total += row.count;
        });

        return jsonResponse({hourly: hourlyData});
    } catch (err) {
        return errorResponse(err.message);
    }
}, 60)); // 趋势图缓存 60秒

// Top Artists
router.get('/api/dashboard/top-artists/:type', async (req, env) => {
    // 带有参数的路由可以也缓存，但 key 要注意。Cache API 默认用 URL 做 key，所以不同 type 不同 limit 都会有不同缓存，没问题。
    return withCache(req, env, async (req, env) => {
        try {
            const db = env.DB;
            const url = new URL(req.url);
            const limit = parseInt(url.searchParams.get("limit") || "10");
            const type = req.params.type;

            let query = "";
            if (type === "tracks") {
                query = "SELECT artist, COUNT(*) as track_count FROM tracks GROUP BY artist ORDER BY track_count DESC LIMIT ?";
            } else {
                query = "SELECT artist, SUM(play_count) as play_count FROM tracks GROUP BY artist ORDER BY play_count DESC LIMIT ?";
            }
            const {results} = await db.prepare(query).bind(limit).all();
            return jsonResponse(results);
        } catch (err) {
            return errorResponse(err.message);
        }
    }, 300);
});

// Top Albums
router.get('/api/dashboard/top-albums', async (req, env) => {
    return withCache(req, env, async (req, env) => {
        try {
            const db = env.DB;
            const url = new URL(req.url);
            const limit = parseInt(url.searchParams.get("limit") || "10");
            const days = parseInt(url.searchParams.get("days") || "30");

            let results;
            if (days > 3650) {
                ({results} = await db.prepare("SELECT album, artist, SUM(play_count) as play_count FROM tracks GROUP BY album, artist ORDER BY play_count DESC LIMIT ?").bind(limit).all());
            } else {
                const startDate = new Date();
                startDate.setDate(startDate.getDate() - days);
                const startDateStr = startDate.toISOString();
                ({results} = await db.prepare("SELECT album, album_artist as artist, COUNT(*) as play_count FROM track_play_records WHERE play_time >= ? GROUP BY album, album_artist ORDER BY play_count DESC LIMIT ?").bind(startDateStr, limit).all());
            }
            return jsonResponse(results);
        } catch (err) {
            return errorResponse(err.message);
        }
    }, 300);
});

// Top Genres
router.get('/api/dashboard/top-genres', async (req, env) => {
    return withCache(req, env, async (req, env) => {
        try {
            const db = env.DB;
            const url = new URL(req.url);
            const limit = parseInt(url.searchParams.get("limit") || "10");
            const {results} = await db.prepare("select tg.track_genre_name, tg.track_genre_count, g.name_zh as genre_name_zh, g.play_count as genre_count from (select genre as track_genre_name, sum(play_count) as track_genre_count from tracks where genre != '' group by genre order by track_genre_count desc limit ?) as tg left join genres as g on tg.track_genre_name = g.name").bind(limit).all();
            return jsonResponse(results);
        } catch (err) {
            return errorResponse(err.message);
        }
    }, 300);
});

// Recent Plays - 不建议缓存太久，或者不缓存，看实时性要求。这里设一个极短的缓存或者不缓存。
// 用户只在刷新时才看，所以可以短缓存 10s 防止并发刷。
router.get('/api/recent-plays', async (req, env) => {
    return withCache(req, env, async (req, env) => {
        try {
            const db = env.DB;
            const url = new URL(req.url);
            const limit = parseInt(url.searchParams.get("limit") || "10");
            const offset = parseInt(url.searchParams.get("offset") || "0");
            const {results} = await db.prepare("SELECT artist, album, track, play_time, source FROM track_play_records ORDER BY play_time DESC LIMIT ? OFFSET ?").bind(limit, offset).all();
            return jsonResponse(results);
        } catch (err) {
            return errorResponse(err.message);
        }
    }, 10);
});

// Track Details
router.get('/api/track', async (req, env) => {
    return withCache(req, env, async (req, env) => {
        try {
            const db = env.DB;
            const url = new URL(req.url);
            const artist = url.searchParams.get("artist");
            const trackName = url.searchParams.get("trackName");
            if (!artist || !trackName) return errorResponse("Missing parameters", 400);

            const track = await db.prepare("SELECT * FROM tracks WHERE artist = ? AND track = ? LIMIT 1").bind(artist, trackName).first();
            if (!track) return errorResponse("Track not found", 404);
            return jsonResponse(track);
        } catch (err) {
            return errorResponse(err.message);
        }
    }, 3600); // 详情页可以缓存很久
});

// Track Play Counts (Total)
router.get('/api/track-play-counts', async (req, env) => {
    return withCache(req, env, async (req, env) => {
        try {
            const db = env.DB;
            const url = new URL(req.url);
            const limit = parseInt(url.searchParams.get("limit") || "10");
            const offset = parseInt(url.searchParams.get("offset") || "0");
            const {results} = await db.prepare("SELECT artist, album, track, play_count FROM tracks ORDER BY play_count DESC LIMIT ? OFFSET ?").bind(limit, offset).all();
            return jsonResponse(results);
        } catch (err) {
            return errorResponse(err.message);
        }
    }, 300);
});

// Track Play Counts (Period)
router.get('/api/track-play-counts/period', async (req, env) => {
    return withCache(req, env, async (req, env) => {
        try {
            const db = env.DB;
            const url = new URL(req.url);
            const limit = parseInt(url.searchParams.get("limit") || "10");
            const offset = parseInt(url.searchParams.get("offset") || "0");
            const period = url.searchParams.get("period") || "week";

            let days = 7;
            if (period === "month") days = 30; else if (period === "year") days = 365;
            const startDate = new Date();
            startDate.setDate(startDate.getDate() - days);
            const startDateStr = startDate.toISOString();

            const {results} = await db.prepare("SELECT artist, album, track, COUNT(*) as play_count FROM track_play_records WHERE play_time >= ? GROUP BY artist, album, track ORDER BY play_count DESC LIMIT ? OFFSET ?").bind(startDateStr, limit, offset).all();
            return jsonResponse(results);
        } catch (err) {
            return errorResponse(err.message);
        }
    }, 300);
});

// Unscrobbled Records - 这个比较实时，但也可以稍微缓存一下
router.get('/api/unscrobbled-records/count', async (req, env) => {
    return withCache(req, env, async (req, env) => {
        try {
            const db = env.DB;
            const {results} = await db.prepare("SELECT COUNT(*) as count FROM track_play_records WHERE scrobbled = 0").first();
            return jsonResponse({count: results?.count || 0});
        } catch (err) {
            return errorResponse(err.message);
        }
    }, 10);
});

router.get('/api/unscrobbled-records', async (req, env) => {
    return withCache(req, env, async (req, env) => {
        try {
            const db = env.DB;
            const url = new URL(req.url);
            const limit = parseInt(url.searchParams.get("limit") || "10");
            const offset = parseInt(url.searchParams.get("offset") || "0");
            const {results} = await db.prepare("SELECT * FROM track_play_records WHERE scrobbled = 0 ORDER BY play_time DESC LIMIT ? OFFSET ?").bind(limit, offset).all();
            return jsonResponse(results);
        } catch (err) {
            return errorResponse(err.message);
        }
    }, 10);
});

// Stubs
router.post('/api/unscrobbled-records/sync', () => jsonResponse({message: "Not supported"}));
router.post('/api/favorite', () => jsonResponse({message: "Not supported"}));


export default {
    fetch: (request, env, ctx) => router.handle(request, env, ctx)
};
