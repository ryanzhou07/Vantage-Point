import { notFound } from 'next/navigation';
import { FeatureUnavailable } from '../../../components/dashboard/feature-unavailable';
import { destinationFor, destinations } from '../../../components/dashboard/navigation';

export function generateStaticParams() {
  return destinations.map((destination) => ({ section: destination.slug }));
}

export default async function DashboardSectionPage({ params }: { params: Promise<{ section: string }> }) {
  const { section } = await params;
  const destination = destinationFor(section);

  if (!destination) notFound();

  return <FeatureUnavailable destination={destination} />;
}
