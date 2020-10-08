import React, {useEffect} from 'react';
import {useHistory} from 'react-router-dom';
// import brandImg from './brandImgColor.svg';
import {
  Card,
  CardHeader,
  CardBody,
} from '@patternfly/react-core';
import {GithubIcon} from '@patternfly/react-icons';
import checkAuthentication from '../redux/Actions/CheckAuthAction';
import GitHubLogin from 'react-github-login';
import {API_URL} from '../../constants';
import {GH_CLIENT_ID} from '../../constants';

const Login: React.FC = () => {
  const history = useHistory();
  const onSuccess = (response) => {
    const authorizeToken = response.code.toString();
    fetch(`${API_URL}/auth/login?code=${authorizeToken}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    })
      .then((res) => res.json())
      .then((response) => {
        localStorage.setItem('token', response.token);
        checkAuthentication();
        history.push('/');
        window.location.reload();
      },
      );
  };
  const onFailure = (error) => {
    console.log(error);
  };
  useEffect(() => {
    // console.log(document.getElementsByTagName('button'));
    document.getElementsByTagName('button')[1].style.backgroundColor =
      '#1e66cc';
    document.getElementsByTagName('button')[1].style.padding = '0.3em';
    document.getElementsByTagName('button')[1].style.width = '50%';
    document.getElementsByTagName('button')[1].style.color = 'white';
  }, []);
  return (
    <div>
      <Card style={{maxWidth: '30em', margin: 'auto'}}>
        <CardHeader style={{
          fontSize: '2em', marginBottom: 0,
          textAlign: 'center',
        }}>
          <GithubIcon size="lg" />
        </CardHeader>
        <CardBody style={{textAlign: 'center'}}>
          <GitHubLogin clientId={GH_CLIENT_ID}
            redirectUri=""
            onSuccess={onSuccess}
            onFailure={onFailure}
            id="1"
          />

        </CardBody>

      </Card>
    </div>
  );
};

// Handle mapStateToProps Here
export default Login;
