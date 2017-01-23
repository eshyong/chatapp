$(document).ready(function() {
    $('#register-button').click(login);
    function login(event) {
        event.preventDefault();
        var $register = $('#register'),
            $username = $register.find('.username').val(),
            $password = $register.find('.password').val(),
            $confirmation = $register.find('.confirmation').val();

        if ($password.length >= 72) {
            // bcrypt only accepts passwords 72 characters or less
            console.log('Password too long');
        }

        if ($password !== $confirmation) {
            console.log('Passwords do not match');
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
