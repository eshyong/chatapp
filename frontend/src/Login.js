import React, { Component } from 'react';

class LoginForm extends Component {
  constructor(props) {
    super(props);
    this.state = {
      userName: '',
      password: ''
    };
  }

  onKeyUp = (event) => {
    // Bind an element's className to its value, and call setState
    this.setState({ [event.target.className]: event.target.value });
  };

  handleLogin = (event) => {
    event.preventDefault();
    console.log('login');
    this.props.clearError();

    if (!this.state.userName || !this.state.password) {
      this.props.showError('Please make sure all fields are filled in');
      return;
    }

    fetch('/login', {
      method: 'POST',
      body: JSON.stringify({
        userName: this.state.userName,
        password: this.state.password
      }),
      headers: {
        accept: 'text/plain',
        'Content-Type': 'application/json',
      },
      credentials: 'same-origin'
    })
    .then((response) => {
      if (response.ok) {
        // Success, redirect to home page
        console.log('Logged in');
        window.location.href = '/';
      } else {
        // Failure
        response.text().then(this.props.showError);
      }
    });
  };

  render() {
    return (
      <form className="LoginForm" style={this.props.style} onSubmit={this.handleLogin}>
        <h2>Login here</h2>
        <input className="userName" type="text" placeholder="User name" onKeyUp={this.onKeyUp}/>
        <input className="password" type="password" placeholder="Password" onKeyUp={this.onKeyUp}/>
        <input type="submit" value="Login"/>
      </form>
    );
  }
}

class RegisterForm extends Component {
  constructor(props) {
    super(props);
    this.state = {
      userName: '',
      password: '',
      confirmation: ''
    };
  }

  handleRegistration = (event) => {
    event.preventDefault();
    console.log('registration');
    this.props.clearError();

    if (!this.state.userName || !this.state.password || !this.state.confirmation) {
      this.props.showError('Please make sure all fields are filled in');
      return;
    }

    if (this.state.password !== this.state.confirmation) {
      this.props.showError('Passwords do not match');
      return;
    }

    fetch('/register', {
      method: 'POST',
      body: JSON.stringify({
        userName: this.state.userName,
        password: this.state.password
      }),
      credentials: 'same-origin'
    })
    .then((response) => {
      if (response.ok) {
        // Success, redirect to home page
        console.log('Registered');
        window.location.href = '/';
      } else {
        // Failure
        response.text().then(this.props.showError);
      }
    })
  };

  onKeyUp = (event) => {
    // Bind an element's className to its value, and call setState
    this.setState({ [event.target.className]: event.target.value });
  };

  render() {
    return (
      <form className="RegisterForm" style={this.props.style} onSubmit={this.handleRegistration}>
        <h2>Or, if this is your first time, register here</h2>
        <input className="userName" type="text" placeholder="User name" onKeyUp={this.onKeyUp}/>
        <input className="password" type="password" placeholder="Password" onKeyUp={this.onKeyUp}/>
        <input className="confirmation" type="password" placeholder="Confirm password" onKeyUp={this.onKeyUp}/>
        <input type="submit" value="Register"/>
      </form>
    );
  }
}

class Login extends Component {
  constructor(props) {
    super(props);
    this.state = {
      error: false,
      message: ''
    };
  }

  clearError = () => {
    this.setState({
      error: false
    });
  };

  showError = (message) => {
    this.setState({
      error: true,
      message: message
    });
  };

  render() {
    let formStyling = {
      alignItems: 'flex-start',
      display: 'flex',
      flexDirection: 'column'
    };
    let errorStyling = { color: 'red' };

    return (
      <div>
        {this.state.error && (
          <div className="errorMessage" style={errorStyling}>{this.state.message}</div>
        )}
          <LoginForm
            style={formStyling}
            showError={this.showError}
            clearError={this.clearError}
          />
          <RegisterForm
            style={formStyling}
            showError={this.showError}
            clearError={this.clearError}
          />
      </div>
    );
  }
}

export default Login;
