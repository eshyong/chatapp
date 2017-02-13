import React from 'react';
import ReactDOM from 'react-dom';
import { Router, Route, hashHistory, browserHistory, withRouter } from 'react-router';

import App from './App';
import Login from './Login';

import './index.css';

function authenticated() {
  return fetch('/user/authenticated', {credentials: 'same-origin'});
}

function main() {
  let historyType = hashHistory;
  if (process.env.NODE_ENV === 'production') {
    historyType = browserHistory;
  }

  authenticated().then((response) => response.json)
  .then((json) => {
    if (json.authenticated) {

    }
  });
  ReactDOM.render((
    <Router history={historyType}>
      <Route path="/" component={App}/>
      <Route path="/login" component={withRouter(Login)}/>
    </Router>
  ), document.getElementById('root'));
}

main();
