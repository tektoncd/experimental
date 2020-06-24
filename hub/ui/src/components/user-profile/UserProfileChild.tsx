import React from 'react';
import {
  Card,
  CardHead,
  Flex,
  FlexItem,
  TextContent,
  CardActions,
  Button,
} from '@patternfly/react-core';
import {DownloadIcon, Remove2Icon, StarIcon} from '@patternfly/react-icons';
import imgAvatar from '../assets/logo/imgAvatar.png';
import {API_URL} from '../../constants';
import {Link} from 'react-router-dom';

export interface TaskPropObject {
  id: number;
  name: string;
  rating: number;
  downloads: number;
}

export interface TaskProp {
  task: TaskPropObject
}

const UserProfileChild: React.FC<TaskProp> = (props: any) => {
  const deleteFunction = (e: any) => {
    const taskId = e.target.id;

    return fetch(`${API_URL}/resource/${taskId}`, {
      method: 'DELETE',
    })
      .then((response) => response.json())
      .then((data: any) => window.location.reload());
  };
  return (
    <div>
      {
        props.task.map((item: any) => {
          return (

            <Card style={{
              marginLeft: '2em', marginRight: '7em',
              marginTop: '2em', width: '100%', padding: '0',
            }} key="">
              <CardHead>
                <img src={imgAvatar} alt="Task"
                  style={{height: '3em', marginLeft: '2em'}}
                />

                <Flex
                  breakpointMods={[{modifier: 'column', breakpoint: 'lg'}]}>
                  <FlexItem>
                    <TextContent
                      style={{marginLeft: '3em', marginTop: '0.5em'}}>
                      <Link to={'/detail/' + item.id} key="">
                        {item.name}
                      </Link>
                    </TextContent>

                  </FlexItem>
                </Flex>

                <CardActions style={{marginRight: '5em'}}>

                  <DownloadIcon
                    style={{marginRight: '0.2em'}} className="download" />
                  <TextContent className="text">{item.downloads}</TextContent>

                  <StarIcon style={{color: '#484848'}} />
                  <TextContent className="text">{item.rating}</TextContent>

                  <Button id={item.id} variant="danger"
                    style={{marginLeft: '3em'}} type="submit"
                    onClick={deleteFunction}>Delete
                    <Remove2Icon
                      style={{marginLeft: '1em', marginTop: '0.3em'}} />
                  </Button>

                </CardActions>
              </CardHead>

            </Card>
          );
        },
        )}
    </div>
  );
};

export default UserProfileChild;
