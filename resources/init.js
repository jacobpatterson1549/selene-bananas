window.addEventListener("load", () => {
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("/main.wasm?v={{.Version}}"), go.importObject)
        .then(async (result) => {
            await go.run(result.instance);
        });
});
