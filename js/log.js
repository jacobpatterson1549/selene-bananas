var log = {
    info: function (text) {
        this._log("info", text);
    },

    warning: function (text) {
        this._log("warning", text);
    },

    error: function (text) {
        this._log("error", text);
    },

    chat: function (message) {
        this._log("chat", message);
    },

    clear: function () {
        var hasLogElement = document.getElementById("has-log");
        hasLogElement.checked = false;
        var logScrollElement = document.getElementById("log-scroll");
        logScrollElement.innerHTML = "";
    },

    _log: function (cls, text) {
        var hasLogElement = document.getElementById("has-log");
        hasLogElement.checked = true;
        var logItemTemplate = document.getElementById("log-item")
        var logItemElement = logItemTemplate.content.cloneNode(true).children[0];
        logItemElement.className = cls;
        var date = new Date();
        time = this.formatDate(date);
        logItemElement.textContent = time + " : " + text;
        var logScrollElement = document.getElementById("log-scroll");
        logScrollElement.appendChild(logItemElement);
        logScrollElement.scrollTop = logScrollElement.scrollHeight - logScrollElement.clientHeight;
    },

    formatDate: function (date) {
        var hour = date.getHours();
        var minutes = date.getMinutes();
        var seconds = date.getSeconds();
        var time = hour + ":" + (minutes > 9 ? minutes : "0" + minutes) + ":" + (seconds > 9 ? seconds : "0" + seconds);
        return time;
    },
};