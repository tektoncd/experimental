import React, {useState} from 'react';
import '@patternfly/react-core/dist/styles/base.css';
import './index.css';
import {
  Flex,
  FlexItem,
  Tooltip,
} from '@patternfly/react-core';
import {useParams} from 'react-router';
import {connect} from 'react-redux';
import checkAuthentication
  from '../redux/Actions/CheckAuthAction';
import {Link} from 'react-router-dom';
import {fetchTaskName} from
  '../redux/Actions/TaskActionName';
import {API_URL} from '../../constants';
import {StarIcon} from '@patternfly/react-icons';
import {fetchTaskSuccess} from '../redux/Actions/TaskAction';

let updatedRating: number = 0;
const Rating: React.FC = (props: any) => {
  const [stars, setStars] = useState(0);
  const [c, setC] = useState(0);
  const [avgRating, setAvgRating] = useState(0.0);
  const [count, setCount] = useState(0);
  const [onechecked, setOnechecked] = useState(false);
  const [twochecked, setTwochecked] = useState(false);
  const [threechecked, setThreechecked] = useState(false);
  const [fourchecked, setFourchecked] = useState(false);
  const [fivechecked, setFivechecked] = useState(false);

  const {taskId} = useParams();

  if (props.TaskData && (c === 0)) {
    for (let i = 0; i < props.TaskData.length; i++) {
      if (props.TaskData[i].id === Number(taskId)) {
        setC((c) => c + 1);
        setAvgRating(props.TaskData[i].rating.toFixed(1));
      }
    }
  }

  // Display rating for particular user
  if (count === 0) {
    fetch(`${API_URL}/resource/${Number(taskId)}/rating`, {
      method: 'GET',
      headers: new Headers({
        'Accept': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      }),
    }).then((res) => res.json()).then((response) => {
      setStars(Number(response.data.rating));
    });
    setCount((count) => count + 1);
  }
  // for showing number of star given by user
  switch (stars) {
    case 5: setFivechecked(true);
      setStars(7); // setting dummy value to star to
      //  avoid falling infinite render loop
      break;
    case 4: setFourchecked(true);
      setStars(7);
      break;
    case 3: setThreechecked(true);
      setStars(7);
      break;
    case 2: setTwochecked(true);
      setStars(7);
      break;
    case 1: setOnechecked(true);
      setStars(7);
      break;
  }
  let login: any = '';
  const putData = (ratingData: any) => {
    fetch(`${API_URL}/resource/${taskId}/rating`, {
      method: 'PUT',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify(ratingData),
    }).then((res) => res.json())
      .then((response) => {
        console.log(response);
        setAvgRating(response.data.avgRating);
      });
  };

  const updateRating = (event: any) => {
    if (event.target.value !== undefined) {
      switch (Number(event.target.value)) {
        case 1:
          setOnechecked(true);
          setTwochecked(false);
          setThreechecked(false);
          setFourchecked(false);
          setFivechecked(false);
          break;
        case 2:
          setTwochecked(true);
          setThreechecked(false);
          setFourchecked(false);
          setFivechecked(false);
          setOnechecked(false);

          break;
        case 3:
          setThreechecked(true);
          setFourchecked(false);
          setFivechecked(false);
          setOnechecked(false);
          setTwochecked(false);
          break;
        case 4:
          setFourchecked(true);
          setFivechecked(false);
          setOnechecked(false);
          setTwochecked(false);
          setThreechecked(false);
          break;
        case 5:
          setFivechecked(true);
          setOnechecked(false);
          setTwochecked(false);
          setThreechecked(false);
          setFourchecked(false);

          break;
      }
      updatedRating = event.target.value;
      const ratingData = {
        'rating': Number(updatedRating),
      };

      putData(ratingData);
    }
  };
  // for checking user is login or not
  if (props.isAuthenticated === true) {
    login = <form onClick={updateRating}>
      <ul className="rate-area" >
        <input type="radio" id="5-star"
          name="rating" value="5" checked={fivechecked} />
        <label htmlFor="5-star" title="Amazing">
          5 stars</label>
        <input type="radio" id="4-star"
          name="rating" value="4" checked={fourchecked} />
        <label htmlFor="4-star" title="Good">
          4 stars</label>
        <input type="radio" id="3-star"
          name="rating" value="3" checked={threechecked} />
        <label htmlFor="3-star" title="Average">
          3 stars</label>
        <input type="radio" id="2-star" name="rating"
          value="2" checked={twochecked} />
        <label htmlFor="2-star" title="Not Good">
          2 stars</label>
        <input type="radio" id="1-star" name="rating"
          value="1" checked={onechecked} />
        <label htmlFor="1-star" title="Bad">
          1 star</label>
      </ul>
    </form>;
  } else {
    login = <form >
      <Link to="/login">
        <ul className="rate-area" >
          <input className="rate-area" type="radio"
            id="5-star" name="rating" value="5" />
          <label htmlFor="5-star"></label>
          <input type="radio" id="4-star"
            name="rating" value="4" />
          <label htmlFor="4-star" ></label>
          <input type="radio" id="3-star" name="rating"
            value="3" /><label htmlFor="3-star" ></label>
          <input type="radio" id="2-star" name="rating"
            value="2" /><label htmlFor="2-star" ></label>
          <input type="radio" id="1-star" name="rating"
            value="1" /><label htmlFor="1-star" ></label>
        </ul>
      </Link>
    </form>;
  }
  return (
    <Flex breakpointMods={[{modifier: 'column', breakpoint: 'lg'}]}>
      <FlexItem style={{marginLeft: '0.2em', marginTop: '0.2em'}}>


        <Flex breakpointMods={[{modifier: 'row', breakpoint: 'lg'}]}>
          <FlexItem>
            <StarIcon style={{color: '#484848'}} />
          </FlexItem>

          <Tooltip
            content={
              <div>Average Rating</div>
            }
          >

            <FlexItem>
              <div>
                {avgRating}
              </div>
            </FlexItem>

          </Tooltip>

        </Flex>


      </FlexItem>
      <div style={{width: '20em'}}>
        <FlexItem style={{marginRight: '-2em', marginLeft: '-9em'}}>
          {login}
        </FlexItem>
      </div>
    </Flex>
  );
};
const mapStateToProps = (state: any) => {
  return {
    isAuthenticated: state.isAuthenticated.isAuthenticated,
    TaskName: state.TaskName.TaskName,
    TaskData: state.TaskData.TaskData,
  };
};
export default connect(mapStateToProps,
  {checkAuthentication, fetchTaskName, fetchTaskSuccess})(Rating);
