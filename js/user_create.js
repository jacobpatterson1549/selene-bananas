var userCreate = {
    submit: function (event) {
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
    }
};