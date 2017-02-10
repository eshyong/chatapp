import React, { Component } from 'react';

class Login extends Component {
  render() {
    return (
      <div
        style={{
          fontSize: '18px',
          fontFamily: 'Helvetica, sans-serif'
        }}>
        <div className="errorMessage" style={{color: 'red'}}></div>
        <form id="login-form"
          style={{
            alignItems: 'flex-start',
            display: 'flex',
            flexDirection: 'column'
          }}>
          <h2>Login here</h2>
          <input className="username" type="text" placeholder="User name"/>
          <input className="password" type="password" placeholder="Password"/>
          <input id="login-button" type="submit" value="Login"/>
        </form>
        <form id="register-form"
          style={{
            alignItems: 'flex-start',
            display: 'flex',
            flexDirection: 'column'
          }}>
          <h2>Or, if this is your first time, register here</h2>
          <input className="username" type="text" placeholder="User name"/>
          <input className="password" type="password" placeholder="Password"/>
          <input className="confirmation" type="text" placeholder="Confirm password"/>
          <input className="register-button" type="submit" value="Register"/>
        </form>
      </div>
    );
  }
}

export default Login;
