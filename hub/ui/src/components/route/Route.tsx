import React from 'react';
import {Route, BrowserRouter as Router} from 'react-router-dom';
import Detail from '../detail/Detail';
import App from '../main/App';
const routing = (
  <Router>
    <Route path='/detail' component={Detail} />
    <Route exact path='/' component={App}/>
  </Router>
);

export default routing;
