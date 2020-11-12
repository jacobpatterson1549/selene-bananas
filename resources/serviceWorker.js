const cacheName = "cache-{{.Version}}";
const assets = [
    "./favicon.png?v={{.Version}}",
    "./favicon.svg?v={{.Version}}",
    "./manifest.json?v={{.Version}}",
    "./wasm_exec.js?v={{.Version}}",
    "./main.wasm?v={{.Version}}",
];

self.addEventListener("install", event => {
    event.waitUntil(
        caches.open(cacheName).then(cache => {
            return cache.addAll(assets);
        })
    );
});

self.addEventListener("fetch", event => {
    event.respondWith(
        caches.match(event.request).then(response => {
            return response || fetch(event.request);
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