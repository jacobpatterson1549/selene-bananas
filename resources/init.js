window.addEventListener("load", () => {
    if ("serviceWorker" in navigator) {
        navigator.serviceWorker.register("./serviceWorker.js");
    }
    const go = new Go();
    WebAssembly.instantiateStreaming(
            fetch("/main.wasm?v={{.Version}}"),
            go.importObject)
        .then(async (result) => {
            await go.run(result.instance);
        });
});
