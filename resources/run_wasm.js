window.addEventListener("load", () => {
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("/main.wasm?v={{.Version}}"), go.importObject)
        .then(async (result) => {
            await go.run(result.instance);
        });
    // TODO: rename file to init.js
    // if ('serviceWorker' in navigator) {
    //     navigator.serviceWorker.register('/service-worker.js?v={{.Version}}');
    // }
})
