import React from 'react';
import PropTypes from 'prop-types';

export default class UserAttribute extends React.PureComponent {
    static propTypes = {
        id: PropTypes.string.isRequired,
        installs: PropTypes.array.isRequired,
        actions: PropTypes.shape({
            getCloudUserData: PropTypes.func.isRequired,
        }).isRequired,
    };

    componentDidMount() {
        this.props.actions.getCloudUserData(this.props.id);
    }

    render() {
        const installs = this.props.installs;

        if (installs.length === 0) {
            return null;
        }

        const entries = installs.map((install) => (
            <li key={install.ID}>
                <a
                    href={'https://' + install.DNS}
                    target='_blank'
                    rel='noopener noreferrer'
                >
                    {install.Name}
                </a>
            </li>
        ));

        return (
            <div style={style.container}>
                <div>{'Cloud Servers'}</div>
                <ul>
                    {entries}
                </ul>
            </div>
        );
    }
}

const style = {
    container: {
        margin: '5px 0',
    },
};
