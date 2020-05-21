var user = {

    _login: function (jwt) {
        content.setJWT(jwt)
        var parts = jwt.split(".");
        var claims = parts[1];
        var jwtUsernameClaims = atob(claims);
        var user = JSON.parse(jwtUsernameClaims);
        var usernameElements = document.querySelectorAll("input.username");
        for (var i = 0; i < usernameElements.length; i++) {
            usernameElements[i].value = user.sub; // the jwt subscriber
            usernameElements[i].setAttribute("readonly", "readonly");
        }
        var pointsElement = document.querySelector("input.points");
        pointsElement.value = user.points;
        var lobbyTab = document.getElementById("tab-4");
        lobbyTab.checked = true;
        content.setLoggedIn(true);
        return Promise.resolve();
    },

    _logout: function () {
        content.setLoggedIn(false);
        game.leave();
        var firstUsernameElement = document.querySelector("input.username");
        firstUsernameElement.setAttribute("readonly", false);
        var loginTab = document.getElementById("tab-1");
        loginTab.checked = true;
        return Promise.resolve();
    },

    _storePassword: function (form) {
        if (window.PasswordCredential) {
            var c = new PasswordCredential(form);
            return navigator.credentials.store(c)
        }
        return Promise.resolve();
    },

    request: function (event) {
        event.preventDefault();
        var form = event.target;
        var method = form.method;
        var url = form.action;
        var host = window.location.host;
        var hostIndex = url.indexOf(host);
        var urlSuffixIndex = host.length + hostIndex;
        var urlSuffix = url.substring(urlSuffixIndex);
        var successPromise;
        switch (urlSuffix) {
            case "/user_create":
                if (content.isLoggedIn()) {
                    content.setErrorMessage("already logged in");
                    return;
                }
                successPromise = () =>
                    this._storePassword(form)
                        .then(this._logout);
                break;
            case "/user_delete":
                var result = window.confirm("Are you sure? All accumulated points will be lost");
                if (!result) {
                    return;
                }
                if (!content.isLoggedIn()) {
                    content.setErrorMessage("not logged in");
                    return;
                }
                successPromise = this._logout;
                break;
            case "/user_login":
                if (content.isLoggedIn()) {
                    content.setErrorMessage("already logged in");
                    return;
                }
                successPromise = response =>
                    this._storePassword(form)
                        .then(() => response.text())
                        .then(this._login);
                break;
            case "/user_logout":
                if (!content.isLoggedIn()) {
                    content.setErrorMessage("not logged in");
                    return;
                }
                successPromise = this._logout;
                break;
            case "/user_update_password":
                if (!content.isLoggedIn()) {
                    content.setErrorMessage("not logged in");
                    return;
                }
                successPromise = () => this._storePassword(form)
                    .then(this._logout);
                break;
            case "/ping":
                successPromise = () => Promise.resolve();
                break;
            default:
                content.setErrorMessage("Unknown action: " + url);
                return;
        }
        var data = {
            method: method,
            credentials: 'include',
        };
        var formData = new FormData(form);
        switch (method) {
            case "post":
                data.body = new URLSearchParams(formData);
                break;
            case "get":
                url += "?" + new URLSearchParams(formData);
                break;
        }
        var jwt = content.getJWT();
        if (jwt) {
            data.headers = {
                Authorization: "Bearer " + jwt,
            };
        }
        fetch(url, data)
            .then(response => {
                if (response.status >= 400) {
                    return Promise.reject(response.status + " " + response.statusText);
                } else {
                    content.setErrorMessage('');
                    return successPromise(response)
                        .then(() => {
                            return Promise.resolve();
                        });
                }
            }).catch(err => {
                content.setErrorMessage(err);
            });
    },

    validatePassword: function (event) {
        var confirmPasswordElement = event.target;
        var confirmPasswordLabelElement = confirmPasswordElement.parentElement;
        var parentFormElement = confirmPasswordLabelElement.parentElement;
        var passwordElement = parentFormElement.querySelector("label>input.password1");
        if (passwordElement.value != confirmPasswordElement.value) {
            confirmPasswordElement.setCustomValidity("Please enter the same password.");
        } else {
            confirmPasswordElement.setCustomValidity("");
        }
    },

    init: function () {
        var confirmPasswordElements = document.querySelectorAll("label>input.password2");
        for (var i = 0; i < confirmPasswordElements.length; i++) {
            confirmPasswordElements[i].onchange = this.validatePassword;
        }
    }
};
