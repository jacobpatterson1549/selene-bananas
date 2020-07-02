window.addEventListener("load", () => {
    // lod WebAssembly
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("/main.wasm?v={{.Version}}"), go.importObject)
        .then(async (result) => {
            await go.run(result.instance);
        });
    // register service worker
    // if ('serviceWorker' in navigator) {
    //     navigator.serviceWorker.register('/service-worker.js?v={{.Version}}');
    // }
})
