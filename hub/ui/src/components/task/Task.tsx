/* eslint-disable max-len */
/* eslint-disable react/prop-types */
import React from 'react';
import '@patternfly/react-core/dist/styles/base.css';
import TimeAgo from 'javascript-time-ago';
import en from 'javascript-time-ago/locale/en';
import {
  Link,
} from 'react-router-dom';
import './index.css';
import {
  Card,
  Badge,
  GalleryItem,
  TextContent,
  CardHead,
  CardHeader,
  CardFooter,
  CardBody,
  CardActions,
  FlexItem,
  Flex,
  FlexModifiers,
} from '@patternfly/react-core';
import {
  StarIcon,
  BuildIcon,
  DomainIcon,
  CatIcon,
  CertificateIcon,
  UserIcon,
} from '@patternfly/react-icons';
export interface TaskPropObject {
  name: string;
  description: string;
  rating: number;
  catalog: string;
  downloads: number;
  yaml: string;
  tags: [];
  lastUpdatedAt: string;
  latestVersion: string;
  displayName: string
}

export interface TaskProp {
  task: TaskPropObject
}

// eslint-disable-next-line
const Task: React.FC<TaskProp> = (props: any) => {
  const tempArr: any = [];
  if (props.task.tags != null) {
    props.task.tags.forEach((item: any) => {
      tempArr.push(item.name);
    });
  } else {
    tempArr.push([]);
  }

  TimeAgo.addLocale(en);

  // Create relative date/time formatter.
  const timeAgo = new TimeAgo('en-US');

  const catalogDate = new Date(props.task.lastUpdatedAt);

  const diffDays = timeAgo.format(catalogDate.getTime() - 60 * 1000);


  // for verification status of resources
  let verifiedStatus: any;
  if (props.task) {
    if (props.task.catalog.type.toLowerCase() === 'official') {
      verifiedStatus = <div className="vtask" >
        <CatIcon size="md" color='#484848'
          style={{width: '2em', height: '2em'}} />
      </div>;
    }
    if (props.task.catalog.type.toLowerCase() === 'verified') {
      verifiedStatus = <div className="vtask" >
        <CertificateIcon size="md" color='#484848' />
      </div>;
    }
    if (props.task.catalog.type.toLowerCase() === 'community') {
      verifiedStatus = <div className="vtask" >
        <UserIcon size="md" color='#484848' />
      </div>;
    }
  }

  // }
  // for adding icon to task and pipeline
  let resourceIcon: React.ReactNode;
  if (props.task.type.toLowerCase() === 'task') {
    resourceIcon = <BuildIcon
      style={{
        width: '2em', height: '2em',
        verticalAlign: '-0.2em',
      }} color="#484848" />;
  } else {
    resourceIcon = <DomainIcon
      style={{
        width: '2em',
        height: '2em', verticalAlign: '-0.2em',
      }}
      color="#484848"
    />;
  };


  // Display name
  let displayName = '';
  if (props.task.displayName === '') {
    displayName = props.task.name;
  } else {
    displayName = props.task.displayName.replace(/(^\w|\s+\w){1}/g, ((str) => {
      return str.toUpperCase();
    }));
  }

  return (
    <GalleryItem>
      <Link to={'/detail/' + props.task.id}>
        <Card className="card" isHoverable
          style={{marginBottom: '1em', borderRadius: '0.5em'}}>

          <CardHead>

            <Flex breakpointMods={[{modifier: 'row', breakpoint: 'lg'}]}>

              <FlexItem style={{marginRight: '1em'}}>
                {resourceIcon}
              </FlexItem>

              <FlexItem>
                {verifiedStatus}
              </FlexItem>

            </Flex>

            <CardActions className="cardActions">

              <StarIcon style={{color: '#484848', height: '1.7em', width: '1.7em'}} />
              <TextContent className="text">
                {props.task.rating.toFixed(1)}
              </TextContent>

            </CardActions>
          </CardHead>
          <CardHeader className="catalog-tile-pf-header">

            <Flex>
              <FlexItem>
                <span className="task-heading">
                  {displayName}
                  {/* {props.task.name[0].toUpperCase() + props.task.name.slice(1)} */}
                </span>
              </FlexItem>
              <FlexItem
                breakpointMods={[{modifier: FlexModifiers['align-right']}]}
                style={{marginBottom: '0.5em'}}>
                <span>
                  v{props.task.latestVersion}
                </span>
              </FlexItem>
            </Flex>
          </CardHeader>
          <CardBody className="catalog-tile-pf-body">
            <div className="catalog-tile-pf-description">
              <span>
                {`${props.task.description.substring(0,
                  props.task.description.indexOf('\n'))}`}
              </span>
            </div>

          </CardBody>
          <CardFooter className="catalog-tile-pf-footer">


            <TextContent className="text"
              style={{marginBottom: '1em', marginLeft: '0.2em'}}>
              Updated {diffDays}
            </TextContent>

            <div style={{height: '2em'}}>
              {
                tempArr.map((tag: any) => {
                  return (
                    <Badge style={{
                      marginLeft: '0.2em',
                      marginBottom: '1em',
                    }} key={`badge-${tag}`} className="badge">{tag}</Badge>
                  );
                })
              }
            </div>

          </CardFooter>
        </Card>
      </Link>
    </GalleryItem >
  );
};
export default Task;
