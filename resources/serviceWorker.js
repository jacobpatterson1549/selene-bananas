const cacheName = "cache-{{.Version}}";
const assets = [
    "./favicon.png?v={{.Version}}",
    "./favicon.svg?v={{.Version}}",
    "./manifest.json?v={{.Version}}",
    "./wasm_exec.js?v={{.Version}}",
    "./main.wasm?v={{.Version}}",
    "./network_check.html?v={{.Version}}",
];

self.addEventListener("install", event => {
    event.waitUntil(
        caches.open(cacheName).then(cache => {
            return cache.addAll(assets);
        })
        .catch(error => {
            throw "install: adding assets to cache failed: " + error;
        })
    );
});

self.addEventListener("fetch", event => {
    event.respondWith(
        caches.match(event.request).then(response => {
            return response || fetch(event.request);
        })
        .catch(error => {
            throw "fetch: failed to get " + event.request.url + ": " + error;
        })
    );
});

self.addEventListener("activate", event => {
    event.waitUntil(
        caches.keys().then(cacheNames => {
            return Promise.all(
                cacheNames.map(name => {
                    if (name !== cacheName) {
                        return caches.delete(name);
                    }
                })
            );
        })
    );
});