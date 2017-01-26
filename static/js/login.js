$(document).ready(function() {
    $('#login-button').click(login);
    $('#register-button').click(register);

    function login(event) {
        event.preventDefault();
        var $login = $('#login'),
            $username = $login.find('.username').val(),
            $password = $login.find('.password').val();

        if (!$username || !$password) {
            console.log('Please make sure all forms are filled');
            return;
        }

        $.post('/login', JSON.stringify({
            username: $username,
            password: $password
        })).done(function(data) {
            if (data) {
                console.log(data);
            }
        });
    }

    function register(event) {
        event.preventDefault();
        var $register = $('#register'),
            $username = $register.find('.username').val(),
            $password = $register.find('.password').val(),
            $confirmation = $register.find('.confirmation').val();

        if (!$username || !$password || !$confirmation) {
            console.log('Please make sure all forms are filled');
            return;
        }

        if ($password.length >= 72) {
            // bcrypt only accepts passwords 72 characters or less
            console.log('Password too long');
            return;
        }

        if ($password !== $confirmation) {
            console.log('Passwords do not match');
            return;
        }

        $.post('/register', JSON.stringify({
            username: $username,
            password: $password
        })).done(function(data) {
            if (data) {
                console.log(data);
            }
        });
    }
});
