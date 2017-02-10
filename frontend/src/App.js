import React, { Component } from 'react';

import Chat from './Chat';
import Login from './Login';

function Navbar(props) {
  let navbarItems = props.navbarItems.map((item, index) => {
    return (
      <div key={index} style={{flexGrow: 1}}>
        <a
          style={{
            background: 'lightskyblue',
            border: '1px solid #000',
            borderRadius: '5px',
            color: 'white',
            display: 'block',
            height: '20px',
            padding: '10px',
            textAlign: 'center',
            textDecoration: 'none'
          }}
          href={item.link}
        >
          {item.label}
        </a>
      </div>
    );
  });
  return (
    <div style={{
      display: 'flex',
      flexDirection: 'row',
      width: '30%'
    }}>
      {navbarItems}
    </div>
  )
}

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
    fetch('/api/authenticated').then((response) => {
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
        <Navbar navbarItems={[
          {
            link: '#',
            label: 'Home'
          },
          {
            link: '#login',
            label: 'Login'
          }
        ]}/>
        <Child/>
      </div>
    );
  }
}

export default App;
