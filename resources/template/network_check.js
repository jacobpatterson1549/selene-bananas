if (navigator.onLine) {
    location.href = "/";
} else {
    window.addEventListener("load", () => {
        var offlineElement = document.querySelector(".offline");
        offlineElement.removeAttribute("hidden");
    });
}