window.addEventListener("load", () => {
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("/main.wasm?uuid={{.UUID}}"), go.importObject)
        .then(async (result) => {
            await go.run(result.instance);
        });
})