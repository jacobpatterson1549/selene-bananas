const cacheName = "{{.Name}}-cache-{{.Version}}";
const assets = [
    "./favicon.ico",
    "./favicon.png",
    "./favicon.svg",
    "./manifest.json",
    "./wasm_exec.js",
    "./selene-bananas.wasm",
    "./network_check.html",
    "./robots.txt",
    "./LICENSE",
];
const addSelfEventListener = (type, listener) => {
    self.addEventListener(type, event => {
        if (event.origin && event.origin !== self.origin) {
            return;
        }
        listener(event);
    });
};

addSelfEventListener("install", event => {
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

addSelfEventListener("fetch", event => {
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

addSelfEventListener("activate", event => {
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