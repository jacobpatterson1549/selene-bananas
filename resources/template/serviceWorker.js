const cacheName = "{{.Name}}-cache-{{.Version}}";
const assets = [
    "./favicon.ico",
    "./favicon.png",
    "./favicon.svg",
    "./manifest.json",
    "./wasm_exec.js",
    "./main.wasm",
    "./network_check.html",
    "./robots.txt",
    "./LICENSE",
];

self.addEventListener("install", event => {
    event.waitUntil(
        caches.open(cacheName)
            .then(cache => {
                return cache.addAll(assets);
            })
            .catch(error => {
                throw new Error("install: adding assets to cache failed: " + error);
            })
    );
});

self.addEventListener("fetch", event => {
    event.respondWith(
        caches.match(event.request)
            .then(response => {
                if (response) {
                    return Promise.resolve(response);
                }
                return fetch(event.request);
            })
            .catch(error => {
                throw new Error("fetch: failed to get " + event.request.url + ": " + error);
            })
    );
});

self.addEventListener("activate", event => {
    event.waitUntil(
        caches.keys()
            .then(cacheNames => {
                return Promise.all(cacheNames
                    .filter(name => (name !== cacheName))
                    .map(name => caches.delete(name))
                );
            })
    );
});
