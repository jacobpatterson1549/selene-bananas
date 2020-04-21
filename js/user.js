var user = {

    _login: function (response) {
        response.text()
            .then(jwt => {
                var jwtInput = document.getElementById("jwt");
                jwtInput.value = jwt;
                var parts = jwt.split(".");
                var claims = parts[1];
                var jwtUsernameClaims = atob(claims);
                var user = JSON.parse(jwtUsernameClaims);
                document.getElementById("user-modify-username").value = user.username;
                document.getElementById("user-modify-points").value = user.points;
                var userModifyTab = document.getElementById("tab-3");
                userModifyTab.checked = true;
                content.setLoggedIn(true);
            });
    },

    _logout: function (response) {
        content.setLoggedIn(false);
        game.leave();
        if (response) {
            var loginTab = document.getElementById("tab-1");
            loginTab.checked = true;
        }
    },

    setModifyAction: function (event) {
        var userModifyRadio = event.target;
        var userModifyForm = document.getElementById("user-modify");
        userModifyForm.action = userModifyRadio.value;
    },

    request: function (event) {
        event.preventDefault();
        var form = event.target;
        var method = form.method
        var url = form.action;
        var host = window.location.host;
        var hostIndex = url.indexOf(host);
        var urlSuffixIndex = host.length + hostIndex;
        var urlSuffix = url.substring(urlSuffixIndex);
        var method;
        var successFn;
        switch (urlSuffix) {
            case "/user_create":
                this._logout();
                method = "POST";
                var self = this;
                successFn = function (response) {
                    if (window.PasswordCredential) {
                        var c = new PasswordCredential(event.target);
                        navigator.credentials.store(c);
                    }
                    self._logout(response);
                }
                break;
            case "/user_delete":
                if (content.isLoggedIn()) {
                    _setErrorMessage("not logged in");
                    return;
                }
                method = "DELETE";
                successFn = this._logout;
                break;
            case "/user_login":
                if (content.isLoggedIn()) {
                    _setErrorMessage("already logged in");
                    return;
                }
                method = "POST";
                successFn = this._login;
                // TODO: create websocket
                break;
            case "/user_logout":
                if (content.isLoggedIn()) {
                    _setErrorMessage("not logged in");
                    return;
                }
                method = "GET";
                this._logout();
                return;
            case "/user_update_password":
                if (content.isLoggedIn()) {
                    _setErrorMessage("not logged in");
                    return;
                }
                method = "PUT";
                successFn = this._logout;
                break;
            default:
                this._setErrorMessage("Unknown action: " + url);
                return;
        }
        var formData = new FormData(form);
        var data = {
            body: new URLSearchParams(formData),
            method: method,
        };
        // TODO: add authorization here
        fetch(url, data)
            .then(async response => {
                if (response.status >= 400) {
                    var message = await response.text();
                    return Promise.reject(message);
                } else {
                    content.setErrorMessage('');
                    successFn(response);
                    return Promise.resolve();
                }
            }).catch(err => {
                content.setErrorMessage(err);
            });
    },

    validatePassword: function (event) {
        var confirmPasswordElement = event.target;
        var confirmPasswordLabelElement = confirmPasswordElement.parentElement;
        var parentFormElement = confirmPasswordLabelElement.parentElement;
        var passwordElement = parentFormElement.querySelector("label > input.password1");
        if (passwordElement.value != confirmPasswordElement.value) {
            confirmPasswordElement.setCustomValidity("Please enter the same password.");
        } else {
            confirmPasswordElement.setCustomValidity("");
        }
    },

    init: function () {
        var confirmPasswordElements = document.querySelectorAll("form > label > input.password2");
        for (var i = 0; i < confirmPasswordElements.length; i++) {
            confirmPasswordElements[i].onchange = this.validatePassword;
        }
    }
};

user.init();
