var user = {
    create: function (event) {
        event.preventDefault();
        var form = event.target;
        var formData = new FormData(form);
        var data = new URLSearchParams(formData);
        fetch("/user_create", {
            method: "POST",
            body: form,
        })
        .then(response => {
            if (response.status != 201) {
                Promise.reject();
            }
        })
        .then(() => { // TODO: check if this is needed
            if (window.PasswordCredential) {
                var c = new PasswordCredential(event.target);
                return navigator.credentials.store(c);
            } else {
                return Promise.resolve();
            }
        }).catch(err => {
            var errorMessageDiv = document.getElementById("error-message");
            errorMessageDiv.innerHTML = err;
        });
    },

    login: function (event) {
        event.preventDefault();
        console.log("TODO: login user");
    },

    logout: function (event) {
        console.log("TODO: logout user (close websocket)");
    },

    update_password: function (event) {
        event.preventDefault();
        console.log("TODO: change user password");
    },

    delete: function (event) {
        event.preventDefault();
        console.log("TODO: delete user");
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
