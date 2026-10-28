import { DemoSection } from "../components/home/DemoSection";
import { FeatureSection } from "../components/home/FeatureSection";
import { Footer } from "../components/home/Footer";
import { Hero } from "../components/home/Hero";
import { Navigation } from "../components/home/Navigation";

export default function Home() {
  return <>
    <Navigation />
    <Hero />
    <FeatureSection />
    <DemoSection />
    <Footer />
  </>;
}
