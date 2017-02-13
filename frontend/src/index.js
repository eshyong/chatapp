import React from 'react';
import ReactDOM from 'react-dom';
import { Router, Route, hashHistory, browserHistory, withRouter } from 'react-router';

import ChatApp from './ChatApp';
import Login from './Login';

import './index.css';

function main() {
  let historyType = hashHistory;
  if (process.env.NODE_ENV === 'production') {
    historyType = browserHistory;
  }

  ReactDOM.render((
    <Router history={historyType}>
      <Route path="/" component={ChatApp}/>
      <Route path="/login" component={withRouter(Login)}/>
    </Router>
  ), document.getElementById('root'));
}

main();
