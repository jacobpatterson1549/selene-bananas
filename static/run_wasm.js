const go = new Go();
WebAssembly.instantiateStreaming(fetch("/main.wasm"), go.importObject)
    .then(async (result) => {
        await go.run(result.instance);
    });