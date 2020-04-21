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
                var usernameElements = document.querySelectorAll("input.username");
                for (i = 0; i < usernameElements.length; i++) {
                    usernameElements[i].value = user.username;
                    // usernameElements[i].readonly = true;
                    usernameElements[i].setAttribute("readonly", true);
                }
                document.querySelector("input.points").value = user.points;
                var userModifyTab = document.getElementById("tab-3");
                userModifyTab.checked = true;
                content.setLoggedIn(true);
                // TODO: create websocket
            });
    },

    _logout: function () {
        content.setLoggedIn(false);
        game.leave();
        var firstUsernameElement = document.querySelector("input.username");
        firstUsernameElement.setAttribute("readonly", false);
        var loginTab = document.getElementById("tab-1");
        loginTab.checked = true;
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
        var successFn;
        switch (urlSuffix) {
            case "/user_create":
                if (content.isLoggedIn()) {
                    content.setErrorMessage("already logged in");
                    return;
                }
                this._logout();
                successFn = function (response) {
                    if (window.PasswordCredential) {
                        var c = new PasswordCredential(form);
                        navigator.credentials.store(c);
                    }
                }
                break;
            case "/user_delete":
                if (!content.isLoggedIn()) {
                    content.setErrorMessage("not logged in");
                    return;
                }
                successFn = this._logout;
                break;
            case "/user_login":
                if (content.isLoggedIn()) {
                    content.setErrorMessage("already logged in");
                    return;
                }
                successFn = this._login;
                break;
            case "/user_logout":
                if (!content.isLoggedIn()) {
                    content.setErrorMessage("not logged in");
                    return;
                }
                successFn = this._logout;
                break;
            case "/user_update_password":
                if (!content.isLoggedIn()) {
                    content.setErrorMessage("not logged in");
                    return;
                }
                successFn = this._logout;
                break;
            default:
                content.setErrorMessage("Unknown action: " + url);
                return;
        }
        var data = {
            method: method,
        };
        if (method === "post") {
            var formData = new FormData(form);
            data.body = new URLSearchParams(formData);
        }
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
