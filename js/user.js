var user = {

    _logout: function () {
        content.setLoggedIn(false);
        game.leave();

    },

    request: function (event, ) {
        event.preventDefault();
        var form = event.target;
        var method = form.method
        var url = form.action;
        var formData = new FormData(form);
        var params = new URLSearchParams(formData);
        var data = {
            method: method,
        };
        var host = window.location.host;
        var hostIndex = url.indexOf(host);
        var urlSuffixIndex = host.length + hostIndex;
        var urlSuffix = url.substring(urlSuffixIndex);
        var successFn;
        switch (urlSuffix) {
            case "/user_create":
                this._logout();
                successFn = function () {
                    if (window.PasswordCredential) {
                        var c = new PasswordCredential(event.target);
                        return navigator.credentials.store(c);
                    } else {
                        return Promise.resolve();
                    }
                }
                break;
            case "/user_delete":
                if (content.isLoggedIn()) {
                    _setErrorMessage("not logged in");
                    return;
                }
                successFn = this._logout;
                break;
            case "/user_login":
                if (content.isLoggedIn()) {
                    _setErrorMessage("already logged in");
                    return;
                }
                // TODO: create websocket
                successFn = this._logout;
                break;
            case "/user_logout":
                if (content.isLoggedIn()) {
                    _setErrorMessage("not logged in");
                    return;
                }
                this._logout();
                return;
            case "/user_update_password":
                if (content.isLoggedIn()) {
                    _setErrorMessage("not logged in");
                    return;
                }
                successFn = this._logout;
                break;
            default:
                this._setErrorMessage("Unknown action: " + url);
                return;
        }
        switch (method.toUpperCase()) {
            case "GET":
            case "PUT":
            case "DELETE":
                url += '?';
                url += params;
                break;
            case "POST":
                data.body = params;
                break;
            default:
                content.setErrorMessage("Unknown http method: " + method);
                return;
        }
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
