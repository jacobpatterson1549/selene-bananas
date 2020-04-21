var webSocket;

var content = {
    setLoggedIn: function (state) {
        var loginElement = document.getElementById("login");
        loginElement.checked = state;
    },

    isLoggedIn: function () {
        var loginElement = document.getElementById("login");
        return loginElement.checked;
    },

    setErrorMessage: function (text) {
        var errorMessageDiv = document.getElementById("error-message");
        errorMessageDiv.innerHTML = text;
    },

    getJWT: function () {
        var jwtInput = document.getElementById("jwt");
        return jwtInput.value;
    },

    setJWT: function (jwt) {
        var jwtInput = document.getElementById("jwt");
        jwtInput.value = jwt;
    }
};