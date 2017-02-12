import React, { Component } from 'react';

import Chat from './Chat';
import Login from './Login';

class App extends Component {
  constructor(props) {
    super(props);
    this.state = {
      authenticated: false,
      route: window.location.hash.substr(1)
    };
  }

  componentDidMount() {
    window.addEventListener('hashchange', () => {
      this.setState({
        route: window.location.hash.substr(1)
      });
    });
    fetch('/user/authenticated').then((response) => {
      if (response.ok) {
        return response.json();
      }
      if (response.status === 500) {
        throw new Error('Unable to contact backend server');
      }
      throw new Error(response.statusText);
    }).then((json) => {
      this.setState({
        authenticated: json.authenticated
      });
    }).catch((error) => {
      console.log(error.message);
    });
  }

  render() {
    let Child;
    if (this.state.authenticated) {
      switch (this.state.route) {
        case '':
          Child = Chat;
          break;
        case 'login':
          Child = Login;
          break;
        default:
          break;
      }
    } else {
      Child = Login;
    }

    return (
      <div>
        <Child/>
      </div>
    );
  }
}

export default App;
