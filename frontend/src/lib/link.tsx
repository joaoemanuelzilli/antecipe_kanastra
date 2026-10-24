import { Link as RouterLink } from 'react-router-dom';
import type { ComponentProps } from 'react';
export default function Link({href,...props}:Omit<ComponentProps<typeof RouterLink>,'to'>&{href:string}) {return <RouterLink to={href} {...props}/>;}
