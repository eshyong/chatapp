$(document).ready(function() {
    $('#login-button').click(login);
    $('#register-button').click(register);

    function login(event) {
        hideError();
        event.preventDefault();

        var $login = $('#login'),
            $username = $login.find('.username').val(),
            $password = $login.find('.password').val();

        if (!$username || !$password) {
            showError('Please make sure all forms are filled');
            return;
        }

        $.post('/login', JSON.stringify({
            username: $username,
            password: $password
        })).then(redirectToIndex, handleHttpError);
    }

    function register(event) {
        hideError();
        event.preventDefault();

        var $register = $('#register'),
            $username = $register.find('.username').val(),
            $password = $register.find('.password').val(),
            $confirmation = $register.find('.confirmation').val();

        if (!$username || !$password || !$confirmation) {
            showError('Please make sure all forms are filled');
            return;
        }

        if ($password.length >= 72) {
            // bcrypt only accepts passwords 72 characters or less
            showError('Password too long: must be 72 characters or less');
            return;
        }

        if ($password !== $confirmation) {
            showError('Passwords do not match');
            return;
        }

        $.post('/register', JSON.stringify({
            username: $username,
            password: $password
        })).then(redirectToIndex, handleHttpError);
    }

    function redirectToIndex() {
        window.location = '/';
    }

    function handleHttpError(requestObj, ignored, httpError) {
        if (requestObj.responseText) {
            showError(requestObj.responseText);
        } else {
            showError(httpError);
        }
    }

    function showError(message) {
        $('.error-message').html(message).show();
    }

    function hideError() {
        $('.error-message').hide();
    }
});
