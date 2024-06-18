/* jshint esversion: 8 */
"use strict";

async function sendCommand(target, command) {
    const response = await fetch("/api/v1/isEntitled");
    const data = await response.json();

    if(data.is_entitled !== "true"){
        document.location.href="/unauthorized";
    }

    var xhr = new XMLHttpRequest();
    xhr.open('POST', '/', true);
    xhr.setRequestHeader('Content-Type', 'application/json');

    var payload = {cluster: target, command: command};

    xhr.send(JSON.stringify(payload));
    setTimeout(document.location.href='/', 1000);
}

var reloading;

function checkReloading() {
    if (window.location.hash === "#autoreload") {
        reloading = setTimeout("window.location.reload();", 30000);
        document.getElementById("autorefreshSwitch").checked = true;
    }
}

function toggleAutoReload(cb) {
    if (cb.checked) {
        window.location.replace("#autoreload");
        reloading = setTimeout("window.location.reload();", 30000);
    } else {
        window.location.replace("/");
        clearTimeout(reloading);
    }
}

window.onload = checkReloading;

async function Validate() {
    // Fetch all the forms we want to apply custom Bootstrap validation styles to
    var forms = document.querySelectorAll('.needs-validation');

    // Loop over them and prevent submission
    Array.prototype.slice.call(forms)
    .forEach(function (form) {
        form.addEventListener('submit', function (event) {
            if (!form.checkValidity()) {
                event.preventDefault();
                event.stopPropagation();
            }

            form.classList.add('was-validated');
        }, false);
    });
}

async function IsEntitled() {
    const response = await fetch("/api/v1/isEntitled");
    const data = await response.json();

    if(data.is_entitled !== "true"){
        document.location.href="/unauthorized";
    }
}